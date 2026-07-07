package main

import (
	"context"
	"log"
	"log/slog"
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

	// Register `atask admin ...` (Task 15) before dispatch so the subcommand set
	// is complete both for the serve-default heuristic below and for cobra's own
	// dispatch. Must happen before app.Start(): PocketBase.Execute() only
	// bootstraps ahead of dispatch when the requested subcommand is already
	// registered on RootCmd — see registerAdminCommands' doc in admin_commands.go.
	registerAdminCommands(app)

	// Default to `serve` only when the user passed no subcommand of their own.
	// Real CLI subcommands (`atask serve|superuser|migrate|admin ...`) must reach
	// cobra untouched, so we only inject the serve defaults when os.Args names no
	// registered subcommand — matched against the actual command set rather than
	// a positional heuristic, so neither a flag value (`atask --http :9000`) nor a
	// subcommand after a global flag (`atask --dir /d admin ...`) is misclassified.
	if !hasSubcommand(os.Args[1:], registeredCommandNames(app)) {
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

		// Task 22: warn operators (rather than silently appearing to lose data)
		// when pre-multi-user rows (user_id = '') are still present. After Task 6
		// enforces user-scoped filtering, such rows are invisible to every user
		// until claimed via `atask admin assign-data --to <user-id>`.
		if counts, err := store.OrphanCounts(context.Background(), db.DB); err != nil {
			slog.Warn("orphan check failed", "err", err)
		} else if total := store.OrphanTotal(counts); total > 0 {
			slog.Warn(
				"orphaned single-user data detected",
				"tables", counts,
				"total_rows", total,
				"remediation", "atask admin assign-data --to <user-id>",
			)
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

// registeredCommandNames returns the set of top-level subcommand names (and
// aliases) registered on the root command — e.g. serve, superuser, migrate
// (PocketBase built-ins) and admin (Task 15). registerAdminCommands must run
// before this is called so `admin` is included.
func registeredCommandNames(app *pocketbase.PocketBase) map[string]bool {
	names := make(map[string]bool)
	for _, c := range app.RootCmd.Commands() {
		names[c.Name()] = true
		for _, alias := range c.Aliases {
			names[alias] = true
		}
	}
	return names
}

// hasSubcommand reports whether the user explicitly invoked one of the
// registered subcommands (rather than expecting the default `serve`). It
// matches each argument against the known command set instead of a positional
// heuristic, so neither a flag value (`atask --http :9000` — ":9000" is not a
// command) nor a subcommand preceded by a global flag (`atask --dir /d admin
// ...`) is misclassified. Cobra strips flags before selecting a command, so a
// real subcommand may legitimately appear after flags.
func hasSubcommand(args []string, known map[string]bool) bool {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if known[a] {
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
