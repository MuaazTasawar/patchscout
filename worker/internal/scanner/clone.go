package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
func Cleanup(clonePath string) error {
	if clonePath == "" {
		return nil
	}
	return os.RemoveAll(clonePath)
}
