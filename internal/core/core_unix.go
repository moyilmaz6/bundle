//go:build darwin

package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moyilmaz6/bundle/internal/runner"
)

func Run() error {
	resources, err := resourcesPath()
	if err != nil {
		return err
	}
	if err := runner.RunPackage(resources); err != nil {
		return err
	}
	return nil
}

func resourcesPath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate bundle core executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(executable); err == nil {
		executable = resolved
	}
	return filepath.Clean(filepath.Join(filepath.Dir(executable), "..", "Resources")), nil
}
