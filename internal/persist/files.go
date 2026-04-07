package persist

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func loadJSONFile[T any](path string, target *T) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return &Error{Op: "read", Path: path, Err: err}
	}

	if err := json.Unmarshal(data, target); err != nil {
		return &Error{Op: "decode", Path: path, Err: err}
	}

	return nil
}

func saveJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return &Error{Op: "encode", Path: path, Err: err}
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(path), directoryPermissions); err != nil {
		return &Error{Op: "mkdir", Path: filepath.Dir(path), Err: err}
	}

	// write through a temp file so interrupted saves never leave a partial JSON file behind.
	tempFile, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return &Error{Op: "create temp", Path: path, Err: err}
	}

	tempName := tempFile.Name()
	cleanup := func() {
		_ = tempFile.Close()
		_ = os.Remove(tempName)
	}

	if _, err := tempFile.Write(data); err != nil {
		cleanup()
		return &Error{Op: "write temp", Path: path, Err: err}
	}
	if err := tempFile.Chmod(filePermissions); err != nil {
		cleanup()
		return &Error{Op: "chmod temp", Path: path, Err: err}
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempName)
		return &Error{Op: "close temp", Path: path, Err: err}
	}
	if err := os.Rename(tempName, path); err != nil {
		_ = os.Remove(tempName)
		return &Error{Op: "rename temp", Path: path, Err: err}
	}

	return nil
}
