package scanner

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/MuaazTasawar/patchscout-worker/internal/models"
)

// semgrepResult mirrors the relevant subset of `semgrep --json` output.
type semgrepResult struct {
	Results []struct {
		CheckID string `json:"check_id"`
		Path    string `json:"path"`
		Start   struct {
			Line int `json:"line"`
		} `json:"start"`
		Extra struct {
			Message  string `json:"message"`
			Severity string `json:"severity"`
			Metadata struct {
				OWASP []string `json:"owasp"`
			} `json:"metadata"`
		} `json:"extra"`
	} `json:"results"`
}

// SASTFinding is a single Semgrep hit, ready to become a models.Finding
// once the caller in scan.go attaches scan_job_id / repo_id.
type SASTFinding struct {
	RuleID   string
	FilePath string
	Line     int
	Message  string
	Severity models.FindingSeverity
}

// RunSemgrep shells out to the Semgrep CLI against a cloned repo, using
// the public "auto" ruleset (community rules, no API token required).
// Requires `semgrep` to be installed on the worker's PATH — see the
// Dockerfile / Render build command in the README.
func RunSemgrep(repoPath string) ([]SASTFinding, error) {
	cmd := exec.Command("semgrep", "--config", "auto", "--json", "--quiet", repoPath)

	output, err := cmd.Output()
	if err != nil {
		// Semgrep exits non-zero when findings exist even without a real
		// error — only treat it as fatal if we got no JSON output at all.
		if len(output) == 0 {
			return nil, fmt.Errorf("semgrep execution failed: %w", err)
		}
	}

	var parsed semgrepResult
	if jsonErr := json.Unmarshal(output, &parsed); jsonErr != nil {
		return nil, fmt.Errorf("failed to parse semgrep output: %w", jsonErr)
	}

	var findings []SASTFinding
	for _, r := range parsed.Results {
		findings = append(findings, SASTFinding{
			RuleID:   r.CheckID,
			FilePath: r.Path,
			Line:     r.Start.Line,
			Message:  r.Extra.Message,
			Severity: mapSemgrepSeverity(r.Extra.Severity),
		})
	}

	return findings, nil
}

func mapSemgrepSeverity(sev string) models.FindingSeverity {
	switch sev {
	case "ERROR":
		return models.SeverityHigh
	case "WARNING":
		return models.SeverityMedium
	default:
		return models.SeverityLow
	}
}
