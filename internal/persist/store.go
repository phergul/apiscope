package persist

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultDirName       = "apiscope"
	configFileName       = "config.json"
	environmentsFileName = "environments.json"
	historyFileName      = "history.json"
	directoryPermissions = 0o700
	filePermissions      = 0o600
)

// Store reads and writes durable app state from one filesystem directory.
type Store struct {
	rootDir string
}

// Error reports one typed persistence failure.
type Error struct {
	Op   string
	Path string
	Err  error
}

// NewStore builds a concrete file-backed persistence store.
//
// When rootDir is blank, the store uses the default app config directory.
func NewStore(rootDir string) *Store {
	return &Store{rootDir: rootDir}
}

func (e *Error) Error() string {
	switch {
	case e == nil:
		return ""
	case e.Path == "":
		return fmt.Sprintf("persist %s: %v", e.Op, e.Err)
	default:
		return fmt.Sprintf("persist %s %s: %v", e.Op, e.Path, e.Err)
	}
}

// Unwrap returns the underlying persistence error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

func (s *Store) configPath() (string, error) {
	return s.filePath(configFileName)
}

func (s *Store) environmentsPath() (string, error) {
	return s.filePath(environmentsFileName)
}

func (s *Store) historyPath() (string, error) {
	return s.filePath(historyFileName)
}

func (s *Store) filePath(fileName string) (string, error) {
	rootDir, err := s.rootPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(rootDir, fileName), nil
}

func (s *Store) rootPath() (string, error) {
	if s != nil && s.rootDir != "" {
		return s.rootDir, nil
	}

	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", &Error{Op: "resolve config dir", Err: err}
	}

	return filepath.Join(baseDir, defaultDirName), nil
}
