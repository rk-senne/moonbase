package compile

import (
	"fmt"
	"os"
)

// IsStale checks whether the compiled JSON is stale relative to the source .md file.
// Returns true if the source is newer than the compiled file, or if the compiled file
// does not exist. Returns false if the compiled file is up to date.
func IsStale(agentMDPath, compiledJSONPath string) (bool, error) {
	srcInfo, err := os.Stat(agentMDPath)
	if err != nil {
		return false, fmt.Errorf("stat source %s: %w", agentMDPath, err)
	}

	dstInfo, err := os.Stat(compiledJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil // Missing compiled file = stale
		}
		return false, fmt.Errorf("stat compiled %s: %w", compiledJSONPath, err)
	}

	// Stale if source is newer than compiled
	return srcInfo.ModTime().After(dstInfo.ModTime()), nil
}
