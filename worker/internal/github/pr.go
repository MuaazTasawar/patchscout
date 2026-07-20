package github

import (
	"fmt"

	"github.com/MuaazTasawar/patchscout-worker/internal/models"
)

// DraftPRBody builds the review-ready PR description for a mechanical CVE
// fix (dependency version bump). This is text only — the worker never
// opens a real branch, commit, or PR itself. Actually creating the PR
// happens only after human approval in the review dashboard (Phase 7),
// which calls the GitHub API directly from the Next.js server using the
// draft content stored on the finding.
func DraftPRBody(f models.Finding, vulnID string) string {
	pkg := ""
	if f.PackageName != nil {
		pkg = *f.PackageName
	}
	vulnVersion := ""
	if f.VulnerableVersion != nil {
		vulnVersion = *f.VulnerableVersion
	}
	patchedVersion := ""
	if f.PatchedVersion != nil {
		patchedVersion = *f.PatchedVersion
	}

	return fmt.Sprintf(`## Security fix: bump %s

**Vulnerability:** %s
**Package:** %s
**Current version:** %s
**Patched version:** %s

%s

---
This PR was drafted automatically by [PatchScout](https://github.com/MuaazTasawar/patchscout) after detecting a known vulnerability via OSV.dev. It bumps %s to the first version containing the fix. Please verify the bump doesn't introduce breaking changes before merging.

_This PR was held for manual review before being opened — it was not posted automatically._
`, pkg, vulnID, pkg, vulnVersion, patchedVersion, f.Description, pkg)
}

// SuggestedBranchName returns a deterministic branch name for a CVE fix,
// used both in the draft PR metadata and (if approved) the real branch
// created at approval time.
func SuggestedBranchName(f models.Finding, vulnID string) string {
	pkg := "dependency"
	if f.PackageName != nil {
		pkg = sanitizeForBranch(*f.PackageName)
	}
	return fmt.Sprintf("patchscout/fix-%s-%s", pkg, sanitizeForBranch(vulnID))
}

// DraftIssueBody builds the review-ready GitHub issue description for a
// Semgrep (SAST) finding — these never become auto-drafted PRs since they
// require human judgment about intent, not just a version bump.
func DraftIssueBody(f models.Finding) string {
	filePath := ""
	if f.FilePath != nil {
		filePath = *f.FilePath
	}
	line := 0
	if f.LineNumber != nil {
		line = *f.LineNumber
	}

	location := filePath
	if line > 0 {
		location = fmt.Sprintf("%s:%d", filePath, line)
	}

	return fmt.Sprintf(`## %s

**Location:** %s
**Severity:** %s

%s

---
This issue was drafted automatically by [PatchScout](https://github.com/MuaazTasawar/patchscout) after a static analysis pass flagged a code pattern worth reviewing. It has not been posted automatically — it was held for manual review, and may be a false positive. Please verify before acting on it.
`, f.Title, location, f.Severity, f.Description)
}

func sanitizeForBranch(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			out = append(out, r)
		case r >= 'A' && r <= 'Z':
			out = append(out, r+32) // lowercase
		case r == '/' || r == '.' || r == '_':
			out = append(out, '-')
		}
	}
	return string(out)
}
