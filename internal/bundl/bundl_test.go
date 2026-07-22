package bundl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/moyilmaz6/bundle/internal/manifest"
)

const darwinARM64 = "darwin-arm64"

func TestOpen(t *testing.T) {
	tests := map[string]struct {
		mutate  func(t *testing.T, directory string)
		wantErr bool
	}{
		"valid_package": {},
		"wrong_target": {mutate: func(t *testing.T, directory string) {
			t.Helper()
			writeDescriptor(t, directory, Descriptor{Package: Package{Format: FormatVersion, Target: "windows-amd64", Name: "Example", Version: "1.0.0", ID: "com.example.app"}})
		}, wantErr: true},
		"non_executable_server": {mutate: func(t *testing.T, directory string) {
			t.Helper()
			if err := os.Chmod(filepath.Join(directory, "server"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, wantErr: true},
		"server_symlink": {mutate: func(t *testing.T, directory string) {
			t.Helper()
			server := filepath.Join(directory, "server")
			if err := os.Remove(server); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("/bin/true", server); err != nil {
				t.Fatal(err)
			}
		}, wantErr: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			directory := testPackage(t)
			if tc.mutate != nil {
				tc.mutate(t, directory)
			}
			_, err := Open(directory, darwinARM64)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Open() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func testPackage(t *testing.T) string {
	t.Helper()
	directory := filepath.Join(t.TempDir(), "Example.bundl")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	writeDescriptor(t, directory, Descriptor{Package: Package{Format: FormatVersion, Target: darwinARM64, Name: "Example", Version: "1.0.0", ID: "com.example.app"}})
	runtime := manifest.Runtime{Server: manifest.RuntimeServer{Port: "8080"}, WebView: manifest.WebView{URL: "http://127.0.0.1:8080/"}, Window: manifest.Window{Title: "Example", Width: 800, Height: 600}}
	runtimeData, err := runtime.ToTOML()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, manifest.RuntimeFileName), runtimeData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "server"), []byte("server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "icon.png"), []byte("icon"), 0o644); err != nil {
		t.Fatal(err)
	}
	return directory
}

func writeDescriptor(t *testing.T, directory string, descriptor Descriptor) {
	t.Helper()
	data, err := descriptor.ToTOML()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, FileName), data, 0o644); err != nil {
		t.Fatal(err)
	}
}
