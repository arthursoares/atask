package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/config"
	"github.com/atask/atask/internal/store"
)

// domainTables lists every table migration 005
// (internal/store/migrations/005_multi_user.sql) added a user_id column to:
// 11 domain tables (including join tables) + 2 event tables. Pre-multi-user
// rows in these tables carry user_id = ” and are invisible to every user
// once user-scoped filtering is enforced (Task 6) until claimed via
// `atask admin assign-data`. This list mirrors Task 22's orphan-detection
// startup guard (internal/store/orphan_check.go).
var domainTables = []string{
	"tasks", "projects", "areas", "sections", "tags",
	"locations", "checklist_items", "activities",
	"task_tags", "project_tags", "task_links",
	"delta_events", "domain_events",
}

// registerAdminCommands adds `atask admin create-user` and
// `atask admin assign-data` to app's cobra RootCmd.
//
// This must be called before app.Start() (equivalently app.Execute()).
// PocketBase.Execute() bootstraps the app (runs migrations, creates default
// collections, opens DB connections) synchronously before dispatching to
// RootCmd.Execute() — but only when the requested subcommand is already
// registered at the time Execute() evaluates skipBootstrap(); an unknown
// command short-circuits and skips bootstrap entirely (see
// PocketBase.skipBootstrap in the pocketbase package). Registering here,
// ahead of main()'s app.Start() call, ensures both subcommands are known and
// get a fully bootstrapped app by the time their RunE runs.
func registerAdminCommands(app *pocketbase.PocketBase) {
	adminCmd := &cobra.Command{Use: "admin", Short: "Admin commands"}

	createUserCmd := &cobra.Command{
		Use:          "create-user",
		Short:        "Create a new user",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			email, _ := cmd.Flags().GetString("email")
			name, _ := cmd.Flags().GetString("name")
			role, _ := cmd.Flags().GetString("role")

			// Note: intentionally not using cobra's MarkFlagRequired here.
			// PocketBase.Execute() (invoked via app.Start()) discards the
			// return value of pb.RootCmd.Execute() entirely (see the
			// "// note: leave to the commands to decide whether to print
			// their error" comment in pocketbase.Execute), so an error
			// returned before RunE ever runs — which is how
			// MarkFlagRequired signals a missing flag — would print a
			// message but still leave the process exit code at 0. Validating
			// here and calling os.Exit(1) ourselves is the only way a
			// calling script can detect failure via $?.
			if email == "" {
				fmt.Fprintln(os.Stderr, "Error: --email is required")
				os.Exit(1)
			}

			password, err := readPassword()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: read password: %v\n", err)
				os.Exit(1)
			}
			if strings.TrimSpace(password) == "" {
				fmt.Fprintln(os.Stderr, "Error: password must not be empty")
				os.Exit(1)
			}

			// PocketBase's own migrations (run during Bootstrap, above) create
			// the `users` auth collection without the role/disabled fields this
			// codebase relies on. The `serve` command's OnServe hook adds them
			// (see main.go), but OnServe never fires for this subcommand, so
			// ensure them here too before saving the record — otherwise
			// role/disabled would be silently dropped (Record.Set on a field the
			// collection doesn't define only sets it in memory; see
			// auth.EnsureUserFields's doc comment).
			if err := auth.EnsureUserFields(app); err != nil {
				fmt.Fprintf(os.Stderr, "Error: ensure user fields: %v\n", err)
				os.Exit(1)
			}

			adapter := auth.NewPBAdapter(app)
			user, err := adapter.CreateUser(email, password, name, role)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Created user: %s (%s) role=%s\n", user.Email, user.ID, user.Role)
			return nil
		},
	}
	createUserCmd.Flags().String("email", "", "User email (required)")
	createUserCmd.Flags().String("name", "", "User name")
	createUserCmd.Flags().String("role", "user", "User role (user or admin)")

	assignDataCmd := &cobra.Command{
		Use:          "assign-data",
		Short:        "Assign orphaned single-user data (user_id = '') to a user",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			userID, _ := cmd.Flags().GetString("to")
			if userID == "" {
				fmt.Fprintln(os.Stderr, "Error: --to is required")
				os.Exit(1)
			}

			cfg := config.Load()
			if err := assignOrphanedData(cfg.DataDir+"/atask.db", userID); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return nil
		},
	}
	assignDataCmd.Flags().String("to", "", "Target user ID (required)")

	adminCmd.AddCommand(createUserCmd, assignDataCmd)
	app.RootCmd.AddCommand(adminCmd)
}

// readPassword prompts for a password on stdout and reads it from stdin.
// When stdin is a real terminal, it uses term.ReadPassword so the password
// is never echoed. When stdin is not a terminal (piped input — scripts,
// CI, tests), there is no echo to suppress, so it falls back to a plain
// buffered line read.
func readPassword() (string, error) {
	fmt.Print("Password: ")
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		pw, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", err
		}
		return string(pw), nil
	}

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// assignOrphanedData opens the domain SQLite database at dbPath and, inside
// a single transaction, sets user_id = userID on every row across
// domainTables that currently carries user_id = ” (pre-multi-user data).
func assignOrphanedData(dbPath, userID string) error {
	db, err := store.NewDB(dbPath)
	if err != nil {
		return fmt.Errorf("open domain db: %w", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		return fmt.Errorf("migrate domain db: %w", err)
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	total := int64(0)
	for _, table := range domainTables {
		// #nosec G201 -- table names come from the constant domainTables
		// whitelist above, never from user input.
		query := fmt.Sprintf(`UPDATE %s SET user_id = ? WHERE user_id = ''`, table)
		res, err := tx.Exec(query, userID)
		if err != nil {
			return fmt.Errorf("update %s: %w", table, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("rows affected for %s: %w", table, err)
		}
		if n > 0 {
			fmt.Printf("%s: %d rows assigned\n", table, n)
		}
		total += n
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	fmt.Printf("Assigned all orphaned data (%d rows) to user %s\n", total, userID)
	return nil
}
