package models

import "time"

// ScanJobStatus mirrors the `status` check constraint on scan_jobs.
type ScanJobStatus string

const (
	StatusQueued   ScanJobStatus = "queued"
	StatusCloning  ScanJobStatus = "cloning"
	StatusScanning ScanJobStatus = "scanning"
	StatusComplete ScanJobStatus = "completed"
	StatusFailed   ScanJobStatus = "failed"
)

// FindingType mirrors findings.type.
type FindingType string

const (
	FindingCVE  FindingType = "cve"
	FindingSAST FindingType = "sast"
)

// FindingSeverity mirrors findings.severity.
type FindingSeverity string

const (
	SeverityCritical FindingSeverity = "critical"
	SeverityHigh     FindingSeverity = "high"
	SeverityMedium   FindingSeverity = "medium"
	SeverityLow      FindingSeverity = "low"
)

// WebhookPayload is what Supabase's pg_net trigger POSTs to the worker
// on every insert into scan_jobs (see 003_scan_jobs_webhook.sql).
type WebhookPayload struct {
	JobID  string `json:"job_id"`
	RepoID string `json:"repo_id"`
}

// Repo mirrors the `repos` table.
type Repo struct {
	ID          string `json:"id"`
	FullName    string `json:"full_name"`
	HTMLURL     string `json:"html_url"`
	Description string `json:"description"`
	Language    string `json:"language"`
}

// ScanJob mirrors the `scan_jobs` table.
type ScanJob struct {
	ID        string        `json:"id"`
	RepoID    string        `json:"repo_id"`
	Status    ScanJobStatus `json:"status"`
	Error     *string       `json:"error"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// Manifest represents a single dependency manifest found in a cloned repo.
type Manifest struct {
	Ecosystem string // "go", "python", "flutter"
	Path      string // relative path within the clone, e.g. "go.mod"
	Packages  []Package
}

// Package is one dependency extracted from a manifest.
type Package struct {
	Name    string
	Version string
}

// Finding mirrors a row destined for the `findings` table. Built up by the
// scanner and detection packages (Phase 5), then POSTed to the Next.js
// worker-callback route (Phase 6) — the worker itself never writes directly
// to Supabase for findings; only scan_jobs status updates go direct.
type Finding struct {
	ScanJobID         string          `json:"scan_job_id"`
	RepoID            string          `json:"repo_id"`
	Type              FindingType     `json:"type"`
	Severity          FindingSeverity `json:"severity"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	PackageName       *string         `json:"package_name,omitempty"`
	VulnerableVersion *string         `json:"vulnerable_version,omitempty"`
	PatchedVersion    *string         `json:"patched_version,omitempty"`
	FilePath          *string         `json:"file_path,omitempty"`
	LineNumber        *int            `json:"line_number,omitempty"`
	DraftPRBranch     *string         `json:"draft_pr_branch,omitempty"`
	DraftPRBody       *string         `json:"draft_pr_body,omitempty"`
	DraftIssueBody    *string         `json:"draft_issue_body,omitempty"`
	DraftEmailBody    *string         `json:"draft_email_body,omitempty"`
}
