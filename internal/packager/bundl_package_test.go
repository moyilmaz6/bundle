package packager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/moyilmaz6/bundle/internal/bundl"
	"github.com/moyilmaz6/bundle/internal/manifest"
)

func TestPackageBundl(t *testing.T) {
	request := testRequest(t, TargetMacOS)
	artifact, err := PackageBundl(request.Manifest, "darwin-arm64")
	if err != nil {
		t.Fatalf("PackageBundl() error = %v", err)
	}
	for _, name := range []string{bundl.FileName, manifest.RuntimeFileName, "server", "icon.png"} {
		if _, err := os.Stat(filepath.Join(artifact.Path, name)); err != nil {
			t.Errorf("expected packaged file %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(artifact.Path, "bundle-core")); !os.IsNotExist(err) {
		t.Errorf(".bundl package unexpectedly contains a core: %v", err)
	}
	server, err := os.Stat(filepath.Join(artifact.Path, "server"))
	if err != nil {
		t.Fatal(err)
	}
	if server.Mode()&0o111 == 0 {
		t.Errorf("server mode = %v, want executable", server.Mode())
	}
	descriptorData, err := os.ReadFile(filepath.Join(artifact.Path, bundl.FileName))
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := bundl.FromTOML(descriptorData)
	if err != nil {
		t.Fatal(err)
	}
	if err := descriptor.Validate("darwin-arm64"); err != nil {
		t.Fatalf("descriptor validation error = %v", err)
	}
}

func TestPackageBundlReplacesExistingPackage(t *testing.T) {
	request := testRequest(t, TargetMacOS)
	first, err := PackageBundl(request.Manifest, "darwin-arm64")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(first.Path, "stale"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := PackageBundl(request.Manifest, "darwin-arm64")
	if err != nil {
		t.Fatal(err)
	}
	if second.Path != first.Path {
		t.Fatalf("replacement path = %q, want %q", second.Path, first.Path)
	}
	if _, err := os.Stat(filepath.Join(second.Path, "stale")); !os.IsNotExist(err) {
		t.Errorf("stale package file remains: %v", err)
	}
}
