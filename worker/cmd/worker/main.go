package main

import (
	"log"
	"net/http"

	"github.com/MuaazTasawar/patchscout-worker/internal/config"
	"github.com/MuaazTasawar/patchscout-worker/internal/handlers"
	"github.com/MuaazTasawar/patchscout-worker/internal/supabase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	db := supabase.New(cfg.SupabaseURL, cfg.SupabaseServiceKey)

	mux := http.NewServeMux()

	webhookHandler := handlers.NewWebhookHandler(db, cfg.CloneDir, cfg.WorkerCallbackSecret)
	mux.Handle("/webhook/scan", webhookHandler)

	// Phase 8 adds /healthz for Render's health check probe.

	log.Printf("patchscout-worker listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
