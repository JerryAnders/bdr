package workspace

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Detect walks up from cwd looking for a .beads/ directory.
// Returns absolute paths to .beads/ and .beads/bdr-index/.
func Detect(cwd string) (beadsDir, indexDir string, err error) {
	dir := cwd
	for {
		candidate := filepath.Join(dir, ".beads")
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, filepath.Join(candidate, "bdr-index"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", "", fmt.Errorf("no beads project found. Run 'bd init' first.")
}

// EnsureGitignore adds bdr-index/ to .beads/.gitignore, creating the file if absent.
// Idempotent.
func EnsureGitignore(beadsDir string) error {
	path := filepath.Join(beadsDir, ".gitignore")
	const entry = "bdr-index/"

	if data, err := os.ReadFile(path); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			if strings.TrimSpace(scanner.Text()) == entry {
				return nil // already present
			}
		}
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintln(f, entry)
	return err
}
