package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	dirPermissions  = 0o755
	filePermissions = 0o644
)

// Manager centralizes where logbooks live on disk and how files are named.
// I/O responsibilities will grow as the Markdown layer is implemented.
type Manager struct {
	basePath string
}

// NewManager constructs a Manager rooted at the provided directory. If basePath
// is empty, it falls back to ~/.kerja (or another location determined by
// ResolveBasePath).
func NewManager(basePath string) (*Manager, error) {
	var err error
	if basePath == "" {
		basePath, err = ResolveBasePath()
		if err != nil {
			return nil, err
		}
	}
	abs, err := filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}

	return &Manager{basePath: abs}, nil
}

// BasePath returns the root directory storing all log files.
func (m *Manager) BasePath() string {
	return m.basePath
}

// MonthPath resolves the absolute path to the markdown file for the supplied time.
// The file may not exist yet; callers can choose to create it.
func (m *Manager) MonthPath(t time.Time) string {
	yearDir := filepath.Join(m.basePath, fmt.Sprintf("%04d", t.Year()))
	return filepath.Join(yearDir, fmt.Sprintf("%04d-%02d.md", t.Year(), t.Month()))
}

// EnsureMonthFile guarantees the directory tree exists and the month file is
// present with the expected heading. It returns the absolute path to the file.
func (m *Manager) EnsureMonthFile(t time.Time) (string, error) {
	if m == nil {
		return "", errors.New("files.Manager is nil")
	}

	path := m.MonthPath(t)
	if err := os.MkdirAll(filepath.Dir(path), dirPermissions); err != nil {
		return "", fmt.Errorf("create directories: %w", err)
	}

	// Attempt to open the file in append mode; create it if necessary.
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, filePermissions)
	if err != nil {
		return "", fmt.Errorf("open month file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("stat month file: %w", err)
	}

	if info.Size() == 0 {
		if _, err := file.WriteString(monthHeader(t)); err != nil {
			return "", fmt.Errorf("write month header: %w", err)
		}
	}

	return path, nil
}

func monthHeader(t time.Time) string {
	return fmt.Sprintf("# %s %04d\n\n", t.Month().String(), t.Year())
}
