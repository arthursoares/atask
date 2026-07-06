package main

import (
	"log"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"github.com/atask/atask/internal/api"
	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/config"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}

	// DefaultDataDir must be supplied at construction: PocketBase eagerly parses
	// the --dir flag straight from os.Args during bootstrap (bypassing cobra's
	// SetArgs), so injecting --dir via SetArgs is silently ignored. Setting the
	// default here makes cfg.DataDir authoritative for pb_data while still leaving
	// an explicit `--dir` override on the CLI available.
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: cfg.DataDir,
	})

	// Default to `serve` only when the user passed no subcommand of their own.
	// Real CLI subcommands (Task 15: `atask superuser ...`, `atask admin ...`,
	// `atask migrate ...`) must still reach cobra untouched, so we only inject
	// the serve defaults when os.Args carries nothing but flags. --http is parsed
	// by the serve command through cobra, so SetArgs applies it correctly.
	if !hasSubcommand(os.Args[1:]) {
		serveArgs := []string{
			"serve",
			"--http=" + normalizeAddr(cfg.Addr),
		}
		serveArgs = append(serveArgs, os.Args[1:]...)
		app.RootCmd.SetArgs(serveArgs)
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Ensure the domain data directory exists (PocketBase's own pb_data lives
		// here too, but the domain atask.db is a separate SQLite file — spec §1 and
		// controller resolution #4: never share a connection with PocketBase).
		if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
			return err
		}

		// Open + migrate the domain database (side-by-side with pb_data/data.db).
		db, err := store.NewDB(cfg.DataDir + "/atask.db")
		if err != nil {
			return err
		}
		if err := db.Migrate(); err != nil {
			return err
		}

		// Close the domain database when PocketBase shuts down. The DB is opened
		// here in OnServe (not at construction), so its close hook is bound here
		// too. OnTerminate fires on graceful shutdown (SIGINT/SIGTERM).
		app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
			if cerr := db.Close(); cerr != nil {
				log.Printf("closing domain db: %v", cerr)
			}
			return te.Next()
		})

		// Ensure PocketBase's users auth collection carries the role/disabled
		// fields the auth adapter reads/writes (name + avatar ship by default).
		if err := auth.EnsureUserFields(se.App); err != nil {
			return err
		}

		// Auth adapter over PocketBase (used by routing starting in Task 11).
		authProvider := auth.NewPBAdapter(app)

		// Event infrastructure.
		bus := event.NewBus()
		eventStore := event.NewEventStore(db)
		streamManager := event.NewStreamManager(bus)

		// Domain services.
		authService := service.NewAuthService(db, jwtSecret())
		taskSvc := service.NewTaskService(db, eventStore, bus)
		projectSvc := service.NewProjectService(db, eventStore, bus)
		areaSvc := service.NewAreaService(db, eventStore, bus)
		sectionSvc := service.NewSectionService(db, eventStore, bus)
		tagSvc := service.NewTagService(db, eventStore, bus)
		locationSvc := service.NewLocationService(db, eventStore, bus)
		checklistSvc := service.NewChecklistService(db, eventStore, bus)
		activitySvc := service.NewActivityService(db, eventStore, bus)

		// Web admin UI (Task 14) needs process-memory CSRF + session stores.
		// Constructed once here so a single instance is shared across the admin
		// handler and its middleware for the lifetime of the server.
		csrfStore := api.NewCSRFStore()
		sessionStore := api.NewSessionStore()

		// Register the domain routes on PocketBase's router with per-route auth
		// (Task 11) and the AuthProvider-backed /auth handlers (Task 12).
		api.RegisterRoutes(se, api.RoutesDeps{
			DB:            db,
			AuthProvider:  authProvider,
			AuthService:   authService,
			Config:        cfg,
			EventStore:    eventStore,
			Bus:           bus,
			StreamManager: streamManager,
			TaskSvc:       taskSvc,
			ProjectSvc:    projectSvc,
			AreaSvc:       areaSvc,
			SectionSvc:    sectionSvc,
			TagSvc:        tagSvc,
			LocationSvc:   locationSvc,
			ChecklistSvc:  checklistSvc,
			ActivitySvc:   activitySvc,
			CSRFStore:     csrfStore,
			SessionStore:  sessionStore,
		})

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

// hasSubcommand reports whether the provided args contain a non-flag token
// (i.e. a cobra subcommand the user wants to run instead of the default serve).
func hasSubcommand(args []string) bool {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return true
		}
	}
	return false
}

// normalizeAddr adapts a bare ":8080" style ADDR into a host:port PocketBase's
// --http flag accepts.
func normalizeAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "0.0.0.0" + addr
	}
	return addr
}

// jwtSecret is retained for AuthService's constructor signature. AuthService's
// JWT-signing methods (CreateUser/Login/ValidateToken) are dead code as of
// Task 12 — /auth/register, /auth/login, and Bearer-token validation all go
// through the PocketBase AuthProvider now; AuthService only backs API-key
// management (which does not use the JWT secret).
func jwtSecret() string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	return "change-me-in-production"
}
