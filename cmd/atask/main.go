package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/atask/atask/internal/api"
	"github.com/atask/atask/internal/client"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
	"github.com/atask/atask/internal/tui"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := buildRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var flagServer string

func buildRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "atask",
		Short: "atask - task management CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(cmd, args)
		},
	}

	rootCmd.PersistentFlags().StringVar(&flagServer, "server", "http://localhost:8080", "atask server URL")

	serveCmd := buildServeCmd()
	rootCmd.AddCommand(serveCmd)

	return rootCmd
}

func buildServeCmd() *cobra.Command {
	var (
		addr      string
		dbPath    string
		jwtSecret string
	)

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the atask HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(cmd, args)
		},
	}

	serveCmd.Flags().StringVar(&addr, "addr", "", "listen address (env: ADDR, default: :8080)")
	serveCmd.Flags().StringVar(&dbPath, "db", "", "database file path (env: DB_PATH, default: atask.db)")
	serveCmd.Flags().StringVar(&jwtSecret, "jwt-secret", "", "JWT signing secret (env: JWT_SECRET)")

	return serveCmd
}

func runTUI(cmd *cobra.Command, args []string) error {
	c := client.New(flagServer, "")
	ctx := context.Background()

	// Auth priority: env var → stored credentials → interactive prompt
	token := os.Getenv("ATASK_TOKEN")

	if token == "" {
		token, _ = loadStoredToken()
	}

	if token != "" {
		c.SetToken(token)
		// Validate stored token — only clear if we get a definite auth error,
		// not a connection error (server might not be running yet)
		if _, err := c.GetMe(ctx); err != nil {
			// Check if it's a connection error vs auth error
			if isAuthError(err) {
				token = ""
			}
			// If it's a connection error, keep the token and let the TUI handle it
		}
	}

	if token == "" {
		// Run TUI login screen
		loginModel := tui.NewLogin(c)
		p := tea.NewProgram(loginModel)
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("login screen: %w", err)
		}
		result := finalModel.(tui.Login).Result()
		if result == nil {
			return nil // user pressed Esc
		}
		token = result.Token
		c.SetToken(token)
		_ = saveToken(token)
	}

	app := tui.NewApp(c)
	p := tea.NewProgram(app)
	_, err := p.Run()
	return err
}

func credentialsPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	return filepath.Join(configDir, "atask", "credentials.json")
}

func loadStoredToken() (string, error) {
	data, err := os.ReadFile(credentialsPath())
	if err != nil {
		return "", err
	}
	var creds struct {
		Token  string `json:"token"`
		Server string `json:"server"`
	}
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", err
	}
	// Only use token if it's for the same server
	if creds.Server != "" && creds.Server != flagServer {
		return "", fmt.Errorf("token is for a different server")
	}
	return creds.Token, nil
}

func saveToken(token string) error {
	path := credentialsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	creds, _ := json.Marshal(struct {
		Token  string `json:"token"`
		Server string `json:"server"`
	}{Token: token, Server: flagServer})
	return os.WriteFile(path, creds, 0600)
}

func isAuthError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "401") || strings.Contains(s, "unauthorized") || strings.Contains(s, "Unauthorized")
}

func runServe(cmd *cobra.Command, args []string) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Config from flags then environment
	dbPath, _ := cmd.Flags().GetString("db")
	if dbPath == "" {
		dbPath = os.Getenv("DB_PATH")
	}
	if dbPath == "" {
		dbPath = "atask.db"
	}

	jwtSecret, _ := cmd.Flags().GetString("jwt-secret")
	if jwtSecret == "" {
		jwtSecret = os.Getenv("JWT_SECRET")
	}
	if jwtSecret == "" {
		jwtSecret = "change-me-in-production"
	}

	addr, _ := cmd.Flags().GetString("addr")
	if addr == "" {
		addr = os.Getenv("ADDR")
	}
	if addr == "" {
		addr = ":8080"
	}

	// Open database
	db, err := store.NewDB(dbPath)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		return err
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		slog.Error("failed to run migrations", "err", err)
		return err
	}

	// Create event infrastructure
	bus := event.NewBus()
	eventStore := event.NewEventStore(db)
	streamManager := event.NewStreamManager(bus)

	// Create services
	authService := service.NewAuthService(db, jwtSecret)
	areaService := service.NewAreaService(db, eventStore, bus)
	taskService := service.NewTaskService(db, eventStore, bus)
	projectService := service.NewProjectService(db, eventStore, bus)
	sectionService := service.NewSectionService(db, eventStore, bus)
	tagService := service.NewTagService(db, eventStore, bus)
	locationService := service.NewLocationService(db, eventStore, bus)
	checklistService := service.NewChecklistService(db, eventStore, bus)
	activityService := service.NewActivityService(db, eventStore, bus)

	// Create handlers
	authHandler := api.NewAuthHandler(authService)
	areaHandler := api.NewAreaHandler(areaService)
	taskHandler := api.NewTaskHandler(taskService)
	projectHandler := api.NewProjectHandler(projectService)
	sectionHandler := api.NewSectionHandler(sectionService)
	tagHandler := api.NewTagHandler(tagService)
	locationHandler := api.NewLocationHandler(locationService)
	checklistHandler := api.NewChecklistHandler(checklistService)
	activityHandler := api.NewActivityHandler(activityService)
	viewHandler := api.NewViewHandler(db)
	eventsHandler := api.NewEventsHandler(streamManager)
	syncHandler := api.NewSyncHandler(eventStore)

	// Wire router
	handler := api.NewRouter(
		areaHandler,
		taskHandler,
		projectHandler,
		sectionHandler,
		tagHandler,
		locationHandler,
		checklistHandler,
		activityHandler,
		viewHandler,
		eventsHandler,
		syncHandler,
		authHandler,
		authService,
	)

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		slog.Info("starting server", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "err", err)
		return err
	}

	slog.Info("server stopped")
	return nil
}
