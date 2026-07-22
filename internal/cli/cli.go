package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moyilmaz6/bundle/internal/coreassets"
	"github.com/moyilmaz6/bundle/internal/manifest"
	"github.com/moyilmaz6/bundle/internal/packager"
	"github.com/spf13/cobra"
)

func Execute() int {
	if err := NewRootCommand().Execute(); err != nil {
		return 1
	}
	return 0
}

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "bundle",
		Short: "Build desktop applications from Go web servers",

		SilenceUsage: true,
	}
	root.AddCommand(newInitCommand(), newBuildCommand())
	root.AddCommand(newPackCommand())
	return root
}

func newPackCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pack <target>",
		Short: "Package a core-free .bundl application for a target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !supportedTarget(args[0]) {
				return unsupportedTarget(args[0])
			}
			app, err := loadManifest()
			if err != nil {
				return err
			}
			artifact, err := packager.PackageBundl(app, args[0])
			if err != nil {
				return fmt.Errorf("package application: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "packed %s\n", artifact.Path)
			return nil
		},
	}
}

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a Bundle application",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(manifest.FileName); err == nil {
				return fmt.Errorf("%s already exists", manifest.FileName)
			} else if !os.IsNotExist(err) {
				return err
			}

			if err := os.WriteFile(manifest.FileName, manifest.TemplateTOML(), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", manifest.FileName)
			return nil
		},
	}
}

func newBuildCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "build <target>",
		Short: "Build a Bundle application for a target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, architecture, err := packageTarget(args[0])
			if err != nil {
				return err
			}
			app, err := loadManifest()
			if err != nil {
				return err
			}
			core, err := coreassets.Load(args[0])
			if err != nil {
				return err
			}
			artifact, err := packager.Package(packager.Request{
				Manifest:     app,
				Target:       target,
				Architecture: architecture,
				Core:         core,
			})
			if err != nil {
				return fmt.Errorf("package application: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "built %s\n", artifact.Path)
			return nil
		},
	}
}

func supportedTarget(target string) bool {
	return packager.IsSupportedTarget(target)
}

func packageTarget(target string) (packager.Target, string, error) {
	platform, architecture, ok := packager.ParseTarget(target)
	if !ok {
		return "", "", unsupportedTarget(target)
	}
	return platform, architecture, nil
}

func unsupportedTarget(target string) error {
	return fmt.Errorf("unsupported target %q; supported targets are %s", target, strings.Join(packager.SupportedTargets(), ", "))
}

func loadManifest() (manifest.Manifest, error) {
	manifestPath, err := filepath.Abs(manifest.FileName)
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("resolve %s: %w", manifest.FileName, err)
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return manifest.Manifest{}, err
	}
	app, err := manifest.ManifestFromTOML(data)
	if err != nil {
		return manifest.Manifest{}, err
	}
	app = app.ResolvePaths(filepath.Dir(manifestPath))
	if err := app.Validate(); err != nil {
		return manifest.Manifest{}, fmt.Errorf("validate %s: %w", manifest.FileName, err)
	}
	return app, nil
}
