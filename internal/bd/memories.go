package bd

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// Memories shells out to `bd memories --json` and returns the key→value map.
func Memories() (map[string]string, error) {
	cmd := exec.Command("bd", "memories", "--json")
	out, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd memories --json failed: %w", err)
		}
		// exec.Error means the binary wasn't found.
		return nil, fmt.Errorf("'bd' command not found. Install beads first.")
	}

	var result map[string]string
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("failed to parse bd memories output: %w", err)
	}
	return result, nil
}
