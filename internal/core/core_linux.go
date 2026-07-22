//go:build linux

package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moyilmaz6/bundle/internal/runner"
)

func Run() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate bundle core executable: %w", err)
	}
	return runner.RunPackage(filepath.Dir(executable))
}
