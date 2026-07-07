package main

import "testing"

// TestHasSubcommand covers the Codex P2 follow-up: hasSubcommand previously
// returned true if ANY arg didn't start with "-", so `atask --http :9000`
// treated the flag *value* ":9000" as a subcommand and skipped injecting the
// default `serve` subcommand + its --http flag (main.go's os.Args handling).
// A cobra subcommand is always the first token, so only args[0] should be
// consulted.
func TestHasSubcommand(t *testing.T) {
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasSubcommand(tc.args)
			if got != tc.want {
				t.Errorf("hasSubcommand(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}
