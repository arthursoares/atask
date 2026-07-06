package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr             string
	DataDir          string
	BaseURL          string
	RegistrationOpen bool

	// OAuth (empty = disabled)
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string

	// Admin
	PocketBaseAdminUI bool
}

func Load() *Config {
	return &Config{
		Addr:               envOr("ADDR", ":8080"),
		DataDir:            envOr("DATA_DIR", "./pb_data"),
		BaseURL:            envOr("BASE_URL", "http://localhost:8080"),
		RegistrationOpen:   os.Getenv("REGISTRATION_OPEN") == "true",
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		PocketBaseAdminUI:  os.Getenv("POCKETBASE_ADMIN_UI") == "true",
	}
}

func (c *Config) Validate() error {
	if c.DataDir == "" {
		return fmt.Errorf("DATA_DIR is required")
	}
	return nil
}

func (c *Config) EnabledProviders() map[string]bool {
	return map[string]bool{
		"email":  true,
		"google": c.GoogleClientID != "",
		"github": c.GitHubClientID != "",
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
