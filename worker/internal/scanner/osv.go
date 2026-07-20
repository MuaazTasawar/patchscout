package scanner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MuaazTasawar/patchscout-worker/internal/models"
)

const osvBatchURL = "https://api.osv.dev/v1/querybatch"
const osvVulnURL = "https://api.osv.dev/v1/vulns/%s"

// osvEcosystem maps our internal ecosystem names to OSV.dev's ecosystem
// identifiers. Flutter/Dart packages are queried under "Pub".
var osvEcosystem = map[string]string{
	"go":      "Go",
	"python":  "PyPI",
	"flutter": "Pub",
}

type osvQuery struct {
	Package struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	} `json:"package"`
	Version string `json:"version"`
}

type osvBatchRequest struct {
	Queries []osvQuery `json:"queries"`
}

type osvBatchResponseItem struct {
	Vulns []struct {
		ID string `json:"id"`
	} `json:"vulns"`
}

type osvBatchResponse struct {
	Results []osvBatchResponseItem `json:"results"`
}

// osvVulnDetail is the subset of OSV's full vuln record we care about:
// the affected ranges (to determine the patched version) and a summary.
type osvVulnDetail struct {
	ID       string `json:"id"`
	Summary  string `json:"summary"`
	Severity []struct {
		Type  string `json:"type"`
		Score string `json:"score"`
	} `json:"severity"`
	Affected []struct {
		Ranges []struct {
			Type   string `json:"type"`
			Events []struct {
				Introduced string `json:"introduced,omitempty"`
				Fixed      string `json:"fixed,omitempty"`
			} `json:"events"`
		} `json:"ranges"`
	} `json:"affected"`
}

// CVEFinding is a single dependency vulnerability match ready to become a
// models.Finding once we know the scan_job_id / repo_id (added by the
// caller in scan.go).
type CVEFinding struct {
	PackageName       string
	VulnerableVersion string
	PatchedVersion    string // empty if OSV didn't report a fix
	VulnID            string
	Summary           string
	Severity          models.FindingSeverity
	FilePath          string
}

// DetectCVEs batch-queries OSV.dev for every package across all discovered
// manifests, then fetches detail for each hit to determine the patched
// version and severity.
func DetectCVEs(manifests []models.Manifest) ([]CVEFinding, error) {
	type pkgRef struct {
		manifest models.Manifest
		pkg      models.Package
	}

	var queries []osvQuery
	var refs []pkgRef

	for _, m := range manifests {
		ecosystem, ok := osvEcosystem[m.Ecosystem]
		if !ok {
			continue
		}
		for _, pkg := range m.Packages {
			q := osvQuery{Version: pkg.Version}
			q.Package.Name = pkg.Name
			q.Package.Ecosystem = ecosystem
			queries = append(queries, q)
			refs = append(refs, pkgRef{manifest: m, pkg: pkg})
		}
	}

	if len(queries) == 0 {
		return nil, nil
	}

	reqBody, err := json.Marshal(osvBatchRequest{Queries: queries})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(osvBatchURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("osv batch query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osv batch query returned status %d", resp.StatusCode)
	}

	var batchResp osvBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, fmt.Errorf("failed to decode osv batch response: %w", err)
	}

	var findings []CVEFinding

	for i, result := range batchResp.Results {
		if i >= len(refs) || len(result.Vulns) == 0 {
			continue
		}
		ref := refs[i]

		// Only fetch full detail for the first vuln ID per package to keep
		// request volume reasonable in v1; multiple simultaneous CVEs on
		// the same package/version are rare enough to defer.
		detail, err := fetchVulnDetail(result.Vulns[0].ID)
		if err != nil {
			continue // skip this finding rather than failing the whole scan
		}

		patched := extractFixedVersion(detail)
		severity := extractSeverity(detail)

		findings = append(findings, CVEFinding{
			PackageName:       ref.pkg.Name,
			VulnerableVersion: ref.pkg.Version,
			PatchedVersion:    patched,
			VulnID:            detail.ID,
			Summary:           detail.Summary,
			Severity:          severity,
			FilePath:          ref.manifest.Path,
		})
	}

	return findings, nil
}

func fetchVulnDetail(id string) (*osvVulnDetail, error) {
	resp, err := http.Get(fmt.Sprintf(osvVulnURL, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("osv vuln detail returned status %d", resp.StatusCode)
	}

	var detail osvVulnDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, err
	}
	return &detail, nil
}

func extractFixedVersion(detail *osvVulnDetail) string {
	for _, affected := range detail.Affected {
		for _, r := range affected.Ranges {
			for _, event := range r.Events {
				if event.Fixed != "" {
					return event.Fixed
				}
			}
		}
	}
	return ""
}

// extractSeverity does a coarse CVSS-score-to-bucket mapping. OSV doesn't
// always include a parsed numeric score, so this falls back to "medium"
// when severity data is absent rather than guessing.
func extractSeverity(detail *osvVulnDetail) models.FindingSeverity {
	if len(detail.Severity) == 0 {
		return models.SeverityMedium
	}

	// CVSS vector strings aren't parsed here (out of scope for v1) — if a
	// numeric-looking score is present in a "score" field we do a rough
	// bucket; otherwise default to medium. This is intentionally simple
	// and safe to revisit later.
	scoreStr := detail.Severity[0].Score
	switch {
	case len(scoreStr) > 0 && scoreStr[0] == '9':
		return models.SeverityCritical
	case len(scoreStr) > 0 && (scoreStr[0] == '7' || scoreStr[0] == '8'):
		return models.SeverityHigh
	case len(scoreStr) > 0 && (scoreStr[0] == '4' || scoreStr[0] == '5' || scoreStr[0] == '6'):
		return models.SeverityMedium
	default:
		return models.SeverityLow
	}
}
