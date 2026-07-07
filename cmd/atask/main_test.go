package main

import "testing"

// TestHasSubcommand covers the Codex P2 follow-up: hasSubcommand previously
// returned true if ANY arg didn't start with "-", so `atask --http :9000`
// treated the flag *value* ":9000" as a subcommand and skipped injecting the
// default `serve` subcommand + its --http flag (main.go's os.Args handling).
// A cobra subcommand is always the first token, so only args[0] should be
// consulted.
func TestHasSubcommand(t *testing.T) {
	// The real registered command set: PocketBase built-ins + Task 15's `admin`.
	known := map[string]bool{"serve": true, "superuser": true, "migrate": true, "admin": true}

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"empty", []string{}, false},
		{"serve", []string{"serve"}, true},
		{"http flag with value", []string{"--http", ":9000"}, false},
		{"admin create-user", []string{"admin", "create-user"}, true},
		{"short help flag", []string{"-h"}, false},
		{"dir flag with value", []string{"--dir", "/data"}, false},
		// Codex PR#4 P2: a subcommand after a global flag must still be recognized
		// (previously the args[0]-only heuristic injected `serve` and dropped it).
		{"global flag then admin subcommand", []string{"--dir", "/data", "admin", "assign-data", "--to", "u"}, true},
		{"global flag then serve", []string{"--dir", "/data", "serve"}, true},
		// A flag value that is not a command name is not a subcommand.
		{"flag value resembling a path", []string{"--http", "/tmp/x"}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasSubcommand(tc.args, known)
			if got != tc.want {
				t.Errorf("hasSubcommand(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}
