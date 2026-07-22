package bundl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moyilmaz6/bundle/internal/manifest"
	"github.com/pelletier/go-toml/v2"
)

const (
	FileName      = "bundl.toml"
	FormatVersion = 1
)

const (
	ServerName        = "server"
	ServerNameWindows = "server.exe"
)

type Descriptor struct {
	Package Package `toml:"package"`
}

type Package struct {
	Format  int    `toml:"format"`
	Target  string `toml:"target"`
	Name    string `toml:"name"`
	Version string `toml:"version"`
	ID      string `toml:"id"`
}

func NewDescriptor(app manifest.Manifest, target string) Descriptor {
	return Descriptor{Package: Package{
		Format:  FormatVersion,
		Target:  target,
		Name:    app.App.Name,
		Version: app.App.Version,
		ID:      app.App.ID,
	}}
}

func FromTOML(data []byte) (Descriptor, error) {
	var descriptor Descriptor
	if err := toml.Unmarshal(data, &descriptor); err != nil {
		return Descriptor{}, err
	}
	return descriptor, nil
}

func (d Descriptor) ToTOML() ([]byte, error) {
	return toml.Marshal(d)
}

func (d Descriptor) Validate(target string) error {
	if d.Package.Format != FormatVersion {
		return fmt.Errorf("unsupported .bundl format %d", d.Package.Format)
	}
	if d.Package.Target != target {
		return fmt.Errorf("package target %q is not supported by this runtime", d.Package.Target)
	}
	if d.Package.Name == "" || d.Package.Version == "" || d.Package.ID == "" {
		return fmt.Errorf("package name, version, and id are required")
	}
	return nil
}

func Open(directory, target string) (manifest.Runtime, error) {
	info, err := os.Stat(directory)
	if err != nil {
		return manifest.Runtime{}, fmt.Errorf("inspect package: %w", err)
	}
	if !info.IsDir() {
		return manifest.Runtime{}, fmt.Errorf("package must be a directory")
	}
	if filepath.Ext(directory) != ".bundl" {
		return manifest.Runtime{}, fmt.Errorf("package must use the .bundl extension")
	}
	descriptorData, err := readRegular(filepath.Join(directory, FileName))
	if err != nil {
		return manifest.Runtime{}, fmt.Errorf("read package descriptor: %w", err)
	}
	descriptor, err := FromTOML(descriptorData)
	if err != nil {
		return manifest.Runtime{}, fmt.Errorf("parse package descriptor: %w", err)
	}
	if err := descriptor.Validate(target); err != nil {
		return manifest.Runtime{}, err
	}
	serverName := ServerFileName(target)
	if _, err := readRegular(filepath.Join(directory, serverName)); err != nil {
		return manifest.Runtime{}, fmt.Errorf("read server: %w", err)
	}
	if _, err := readRegular(filepath.Join(directory, "icon.png")); err != nil {
		return manifest.Runtime{}, fmt.Errorf("read icon: %w", err)
	}
	runtimeData, err := readRegular(filepath.Join(directory, manifest.RuntimeFileName))
	if err != nil {
		return manifest.Runtime{}, fmt.Errorf("read runtime configuration: %w", err)
	}
	runtime, err := manifest.RuntimeFromTOML(runtimeData)
	if err != nil {
		return manifest.Runtime{}, fmt.Errorf("parse runtime configuration: %w", err)
	}
	if err := runtime.Validate(); err != nil {
		return manifest.Runtime{}, fmt.Errorf("validate runtime configuration: %w", err)
	}
	serverInfo, err := os.Stat(filepath.Join(directory, serverName))
	if err != nil {
		return manifest.Runtime{}, fmt.Errorf("stat server: %w", err)
	}
	if !isWindowsTarget(target) && serverInfo.Mode()&0o111 == 0 {
		return manifest.Runtime{}, fmt.Errorf("server is not executable")
	}
	return runtime, nil
}

func ServerFileName(target string) string {
	if isWindowsTarget(target) {
		return ServerNameWindows
	}
	return ServerName
}

func isWindowsTarget(target string) bool {
	return strings.HasPrefix(target, "windows-")
}

func readRegular(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("must be a regular file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data, nil
}
