package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ShallowClone clones a repo at depth 1 into <cloneDir>/<jobID>/ and
// returns the absolute path to the clone. Callers are responsible for
// calling Cleanup when done — clones can be several hundred MB and this
// worker may process many repos concurrently.
func ShallowClone(cloneDir, jobID, htmlURL string) (string, error) {
	dest := filepath.Join(cloneDir, jobID)

	if err := os.MkdirAll(cloneDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create clone dir: %w", err)
	}

	cmd := exec.Command("git", "clone", "--depth", "1", htmlURL, dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git clone failed: %w\noutput: %s", err, string(output))
	}

	absDest, err := filepath.Abs(dest)
	if err != nil {
		return "", fmt.Errorf("failed to resolve clone path: %w", err)
	}

	return absDest, nil
}

// Cleanup removes a clone directory once scanning is complete (success or
// failure). Always call this via defer right after ShallowClone succeeds.
//
// On Windows, files inside a freshly-cloned .git directory (especially
// packed objects) can briefly be held open by antivirus real-time scanning
// or the Windows file indexer, causing the first RemoveAll attempt to fail
// with "The process cannot access the file because it is being used by
// another process." This is transient — retrying after a short delay
// almost always succeeds once the handle is released. We also clear any
// read-only attributes git sets on object files, since those alone can
// block deletion independent of the file-lock issue.
func Cleanup(clonePath string) error {
	if clonePath == "" {
		return nil
	}

	clearReadOnly(clonePath)

	var lastErr error
	backoff := 200 * time.Millisecond

	for attempt := 1; attempt <= 5; attempt++ {
		lastErr = os.RemoveAll(clonePath)
		if lastErr == nil {
			return nil
		}
		time.Sleep(backoff)
		backoff *= 2
		clearReadOnly(clonePath) // re-clear in case more locks released
	}

	return fmt.Errorf("failed to remove clone dir after retries: %w", lastErr)
}

// clearReadOnly walks the tree and strips the read-only attribute from
// every file, ignoring errors — this is a best-effort pass, not a
// hard requirement, since the retry loop in Cleanup handles the case
// where a file is still genuinely locked rather than just read-only.
func clearReadOnly(root string) {
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() {
			_ = os.Chmod(path, 0666)
		}
		return nil
	})
}
