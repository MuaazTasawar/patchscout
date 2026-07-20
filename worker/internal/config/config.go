package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds all environment-derived settings for the worker.
type Config struct {
	Port                 string
	SupabaseURL          string
	SupabaseServiceKey   string
	GitHubToken          string
	WorkerCallbackSecret string
	NextCallbackURL      string
	CloneDir             string
}

// Load reads worker/.env (if present) then required env vars. Missing
// required values return a descriptive error instead of panicking, so
// main.go can log and exit cleanly.
func Load() (*Config, error) {
	// Ignore error: .env is optional in production (Render injects env vars
	// directly), but convenient for local dev.
	_ = godotenv.Load()

	cfg := &Config{
		Port:                 getEnvDefault("PORT", "8080"),
		SupabaseURL:          os.Getenv("SUPABASE_URL"),
		SupabaseServiceKey:   os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		GitHubToken:          os.Getenv("GITHUB_TOKEN"),
		WorkerCallbackSecret: os.Getenv("WORKER_CALLBACK_SECRET"),
		NextCallbackURL:      os.Getenv("NEXT_CALLBACK_URL"),
		CloneDir:             getEnvDefault("CLONE_DIR", "./clones"),
	}

	missing := []string{}
	if cfg.SupabaseURL == "" {
		missing = append(missing, "SUPABASE_URL")
	}
	if cfg.SupabaseServiceKey == "" {
		missing = append(missing, "SUPABASE_SERVICE_ROLE_KEY")
	}
	if cfg.WorkerCallbackSecret == "" {
		missing = append(missing, "WORKER_CALLBACK_SECRET")
	}
	if cfg.NextCallbackURL == "" {
		missing = append(missing, "NEXT_CALLBACK_URL")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}

func getEnvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
