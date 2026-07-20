package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/MuaazTasawar/patchscout-worker/internal/models"
	"github.com/MuaazTasawar/patchscout-worker/internal/scanner"
	"github.com/MuaazTasawar/patchscout-worker/internal/supabase"
)

// WebhookHandler receives the pg_net POST fired by the scan_jobs insert
// trigger (see 003_scan_jobs_webhook.sql) and kicks off scanning for that
// job in a goroutine, so the HTTP response returns immediately.
type WebhookHandler struct {
	DB       *supabase.Client
	CloneDir string
	Secret   string
}

func NewWebhookHandler(db *supabase.Client, cloneDir, secret string) *WebhookHandler {
	return &WebhookHandler{DB: db, CloneDir: cloneDir, Secret: secret}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("X-Webhook-Secret") != h.Secret {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload models.WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if payload.JobID == "" || payload.RepoID == "" {
		http.Error(w, "job_id and repo_id are required", http.StatusBadRequest)
		return
	}

	// Fire-and-forget: acknowledge the webhook immediately, scan async.
	// Detection pipeline (Phase 5) and result submission (Phase 6) hook
	// into runScanJob below as they're added.
	go h.runScanJob(payload)

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"accepted":true}`))
}

func (h *WebhookHandler) runScanJob(payload models.WebhookPayload) {
	jobID := payload.JobID
	repoID := payload.RepoID

	log.Printf("[job %s] starting scan for repo %s", jobID, repoID)

	if err := h.DB.UpdateScanJob(jobID, map[string]interface{}{"status": models.StatusCloning}); err != nil {
		log.Printf("[job %s] failed to update status to cloning: %v", jobID, err)
	}

	repoRow, err := h.DB.GetRepoByID(repoID)
	if err != nil {
		h.fail(jobID, "failed to fetch repo: "+err.Error())
		return
	}

	htmlURL, _ := repoRow["html_url"].(string)
	if htmlURL == "" {
		h.fail(jobID, "repo row missing html_url")
		return
	}

	clonePath, err := scanner.ShallowClone(h.CloneDir, jobID, htmlURL)
	if err != nil {
		h.fail(jobID, "clone failed: "+err.Error())
		return
	}
	defer func() {
		if cleanupErr := scanner.Cleanup(clonePath); cleanupErr != nil {
			log.Printf("[job %s] cleanup warning: %v", jobID, cleanupErr)
		}
	}()

	if err := h.DB.UpdateScanJob(jobID, map[string]interface{}{"status": models.StatusScanning}); err != nil {
		log.Printf("[job %s] failed to update status to scanning: %v", jobID, err)
	}

	manifests, err := scanner.DiscoverManifests(clonePath)
	if err != nil {
		h.fail(jobID, "manifest discovery failed: "+err.Error())
		return
	}

	log.Printf("[job %s] found %d manifest(s) in %s", jobID, len(manifests), clonePath)

	// Phase 5 wires DetectCVEs/DetectSAST in here and Phase 6 wires the
	// findings POST-back to Next.js. For now, a repo with manifests but no
	// detection pipeline yet completes cleanly with zero findings.
	if err := h.DB.UpdateScanJob(jobID, map[string]interface{}{"status": models.StatusComplete}); err != nil {
		log.Printf("[job %s] failed to update status to completed: %v", jobID, err)
	}

	log.Printf("[job %s] scan complete", jobID)
}

func (h *WebhookHandler) fail(jobID, reason string) {
	log.Printf("[job %s] FAILED: %s", jobID, reason)
	_ = h.DB.UpdateScanJob(jobID, map[string]interface{}{
		"status": models.StatusFailed,
		"error":  reason,
	})
}

// mustGetenv is a small helper used by main.go; kept here to avoid an
// extra file for one function used at startup only.
func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return v
}
