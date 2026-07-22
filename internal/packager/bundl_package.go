package packager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moyilmaz6/bundle/internal/bundl"
	"github.com/moyilmaz6/bundle/internal/manifest"
)

func PackageBundl(app manifest.Manifest, target string) (Artifact, error) {
	if err := app.Validate(); err != nil {
		return Artifact{}, err
	}
	if !IsSupportedTarget(target) {
		return Artifact{}, fmt.Errorf("unsupported .bundl target %q", target)
	}
	server, err := os.ReadFile(app.Server.Binary)
	if err != nil {
		return Artifact{}, fmt.Errorf("read server binary: %w", err)
	}
	icon, err := os.ReadFile(app.App.Icon)
	if err != nil {
		return Artifact{}, fmt.Errorf("read app icon: %w", err)
	}
	descriptorData, err := bundl.NewDescriptor(app, target).ToTOML()
	if err != nil {
		return Artifact{}, fmt.Errorf("encode package descriptor: %w", err)
	}
	runtimeData, err := app.Runtime().ToTOML()
	if err != nil {
		return Artifact{}, fmt.Errorf("encode runtime configuration: %w", err)
	}

	name := artifactName(app.App.Name)
	packagePath := filepath.Join(app.Output.Path, name+".bundl")
	if err := os.MkdirAll(app.Output.Path, 0o755); err != nil {
		return Artifact{}, err
	}
	stagingPath, err := os.MkdirTemp(app.Output.Path, "."+name+".bundl-")
	if err != nil {
		return Artifact{}, err
	}
	defer os.RemoveAll(stagingPath)
	for _, file := range []struct {
		name string
		data []byte
		mode os.FileMode
	}{
		{name: bundl.FileName, data: descriptorData, mode: 0o644},
		{name: manifest.RuntimeFileName, data: runtimeData, mode: 0o644},
		{name: bundl.ServerFileName(target), data: server, mode: 0o755},
		{name: "icon.png", data: icon, mode: 0o644},
	} {
		if err := writeFile(filepath.Join(stagingPath, file.name), file.data, file.mode); err != nil {
			return Artifact{}, fmt.Errorf("write %s: %w", file.name, err)
		}
	}
	if err := replaceDirectory(stagingPath, packagePath); err != nil {
		return Artifact{}, err
	}
	return Artifact{Path: packagePath}, nil
}

func replaceDirectory(stagingPath, destination string) error {
	backupPath := destination + ".previous"
	if err := os.RemoveAll(backupPath); err != nil {
		return err
	}
	if _, err := os.Lstat(destination); err == nil {
		if err := os.Rename(destination, backupPath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(stagingPath, destination); err != nil {
		if restoreErr := os.Rename(backupPath, destination); restoreErr != nil && !os.IsNotExist(restoreErr) {
			return fmt.Errorf("install package: %w; restore previous package: %v", err, restoreErr)
		}
		return err
	}
	if err := os.RemoveAll(backupPath); err != nil {
		return err
	}
	return nil
}
