//go:build windows

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
	return runner.RunPackage(resources)
}

func resourcesPath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate bundle core executable: %w", err)
	}
	return filepath.Dir(executable), nil
}
