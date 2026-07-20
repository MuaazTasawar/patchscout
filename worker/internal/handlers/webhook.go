package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	patchscoutgithub "github.com/MuaazTasawar/patchscout-worker/internal/github"
	"github.com/MuaazTasawar/patchscout-worker/internal/models"
	"github.com/MuaazTasawar/patchscout-worker/internal/scanner"
	"github.com/MuaazTasawar/patchscout-worker/internal/supabase"
)

// WebhookHandler receives the pg_net POST fired by the scan_jobs insert
// trigger (see 003_scan_jobs_webhook.sql) and kicks off scanning for that
// job in a goroutine, so the HTTP response returns immediately.
type WebhookHandler struct {
	DB              *supabase.Client
	CloneDir        string
	Secret          string
	NextCallbackURL string
	HTTPClient      *http.Client
}

func NewWebhookHandler(db *supabase.Client, cloneDir, secret, nextCallbackURL string) *WebhookHandler {
	return &WebhookHandler{
		DB:              db,
		CloneDir:        cloneDir,
		Secret:          secret,
		NextCallbackURL: nextCallbackURL,
		HTTPClient:      &http.Client{},
	}
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

	result, detErr := scanner.RunDetection(clonePath, manifests)
	if detErr != nil {
		h.fail(jobID, "detection failed: "+detErr.Error())
		return
	}

	findings := result.ToFindings(jobID, repoID)

	// Attach draft PR/issue text before sending to the callback — the
	// worker prepares the text, but never posts anything to GitHub itself.
	for i := range findings {
		if findings[i].Type == models.FindingCVE {
			vulnID := ""
			if len(result.CVEFindings) > i {
				vulnID = result.CVEFindings[i].VulnID
			}
			if findings[i].PatchedVersion != nil {
				body := patchscoutgithub.DraftPRBody(findings[i], vulnID)
				branch := patchscoutgithub.SuggestedBranchName(findings[i], vulnID)
				findings[i].DraftPRBody = &body
				findings[i].DraftPRBranch = &branch
			}
		} else if findings[i].Type == models.FindingSAST {
			body := patchscoutgithub.DraftIssueBody(findings[i])
			findings[i].DraftIssueBody = &body
		}
	}

	if len(findings) > 0 {
		if err := h.postFindings(jobID, findings); err != nil {
			log.Printf("[job %s] failed to post findings to callback: %v", jobID, err)
			// Don't fail the whole job over a callback error — the scan
			// itself succeeded; log loudly so it can be retried/inspected.
		}
	}

	if err := h.DB.UpdateScanJob(jobID, map[string]interface{}{"status": models.StatusComplete}); err != nil {
		log.Printf("[job %s] failed to update status to completed: %v", jobID, err)
	}

	log.Printf("[job %s] scan complete — %d finding(s)", jobID, len(findings))
}

// postFindings sends the completed findings to the Next.js worker-callback
// route, which is the ONLY place findings get written to the database.
// This keeps the human-review gate centralized in one code path.
func (h *WebhookHandler) postFindings(jobID string, findings []models.Finding) error {
	payload, err := json.Marshal(map[string]interface{}{
		"job_id":   jobID,
		"findings": findings,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, h.NextCallbackURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", h.Secret)

	resp, err := h.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("callback returned status %d", resp.StatusCode)
	}
	return nil
}

func (h *WebhookHandler) fail(jobID, reason string) {
	log.Printf("[job %s] FAILED: %s", jobID, reason)
	_ = h.DB.UpdateScanJob(jobID, map[string]interface{}{
		"status": models.StatusFailed,
		"error":  reason,
	})
}
