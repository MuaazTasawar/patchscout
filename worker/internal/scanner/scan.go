package scanner

import (
	"fmt"

	"github.com/MuaazTasawar/patchscout-worker/internal/models"
)

// ScanResult bundles everything a completed scan produced, in a form ready
// to be converted into models.Finding rows once scan_job_id/repo_id are
// known (done by the caller — see handlers/webhook.go in Phase 6).
type ScanResult struct {
	Manifests    []models.Manifest
	CVEFindings  []CVEFinding
	SASTFindings []SASTFinding
}

// RunDetection runs both detection passes (OSV dependency check + Semgrep
// static analysis) against an already-cloned repo. Manifest discovery has
// already happened by the time this is called (Phase 4); this function
// takes the manifests as input so callers can log/inspect them first.
//
// Both passes run independently — a failure in one does not block the
// other, since dependency CVEs and code-pattern bugs are unrelated checks.
func RunDetection(repoPath string, manifests []models.Manifest) (*ScanResult, error) {
	result := &ScanResult{Manifests: manifests}

	cveFindings, cveErr := DetectCVEs(manifests)
	if cveErr != nil {
		// Log-and-continue: OSV being unreachable shouldn't block SAST.
		cveFindings = nil
	}
	result.CVEFindings = cveFindings

	sastFindings, sastErr := RunSemgrep(repoPath)
	if sastErr != nil {
		sastFindings = nil
	}
	result.SASTFindings = sastFindings

	if cveErr != nil && sastErr != nil {
		return result, fmt.Errorf("both detection passes failed: cve=%v sast=%v", cveErr, sastErr)
	}

	return result, nil
}

// ToFindings converts a ScanResult into models.Finding rows, attaching the
// scan_job_id and repo_id that are only known at the call site.
func (r *ScanResult) ToFindings(scanJobID, repoID string) []models.Finding {
	var findings []models.Finding

	for _, cve := range r.CVEFindings {
		pkgName := cve.PackageName
		vulnVersion := cve.VulnerableVersion
		filePath := cve.FilePath

		var patchedVersion *string
		if cve.PatchedVersion != "" {
			p := cve.PatchedVersion
			patchedVersion = &p
		}

		findings = append(findings, models.Finding{
			ScanJobID:         scanJobID,
			RepoID:            repoID,
			Type:              models.FindingCVE,
			Severity:          cve.Severity,
			Title:             fmt.Sprintf("%s: %s affected by %s", cve.VulnID, pkgName, cve.VulnID),
			Description:       cve.Summary,
			PackageName:       &pkgName,
			VulnerableVersion: &vulnVersion,
			PatchedVersion:    patchedVersion,
			FilePath:          &filePath,
		})
	}

	for _, sast := range r.SASTFindings {
		filePath := sast.FilePath
		line := sast.Line

		findings = append(findings, models.Finding{
			ScanJobID:   scanJobID,
			RepoID:      repoID,
			Type:        models.FindingSAST,
			Severity:    sast.Severity,
			Title:       fmt.Sprintf("Semgrep: %s", sast.RuleID),
			Description: sast.Message,
			FilePath:    &filePath,
			LineNumber:  &line,
		})
	}

	return findings
}
