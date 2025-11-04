package files

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	// DefaultDirName defines the folder under the user's home directory.
	DefaultDirName = ".kerja"
)

// ResolveBasePath determines where kerja stores Markdown logs, defaulting to ~/.kerja.
// The location can be overridden by exporting KERJA_HOME.
func ResolveBasePath() (string, error) {
	if override, ok := os.LookupEnv("KERJA_HOME"); ok {
		override = strings.TrimSpace(override)
		if override != "" {
			path, err := normalizePath(override)
			if err != nil {
				return "", err
			}
			return path, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DefaultDirName), nil
}

func normalizePath(input string) (string, error) {
	if strings.HasPrefix(input, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		input = filepath.Join(home, strings.TrimPrefix(input, "~"))
	}
	return input, nil
}
