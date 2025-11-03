package files

import (
	"os"
	"path/filepath"
)

const (
	// DefaultDirName defines the folder under the user's home directory.
	DefaultDirName = ".kerja"
)

// ResolveBasePath determines where kerja stores Markdown logs, defaulting to ~/.kerja.
// Later this will honor environment overrides like KERJA_HOME.
func ResolveBasePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DefaultDirName), nil
}

