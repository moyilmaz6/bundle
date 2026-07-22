package packager

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/moyilmaz6/bundle/internal/manifest"
)

func TestPackageMacOS(t *testing.T) {
	request := testRequest(t, TargetMacOS)
	artifact, err := Package(request)
	if err != nil {
		t.Fatalf("Package() error = %v", err)
	}

	for _, path := range []string{
		filepath.Join(artifact.Path, "Contents", "Info.plist"),
		filepath.Join(artifact.Path, "Contents", "MacOS", "Example-App"),
		filepath.Join(artifact.Path, "Contents", "Resources", "server"),
		filepath.Join(artifact.Path, "Contents", "Resources", manifest.RuntimeFileName),
		filepath.Join(artifact.Path, "Contents", "Resources", "AppIcon.icns"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected packaged file %s: %v", path, err)
		}
	}
	for _, path := range []string{
		filepath.Join(artifact.Path, "Contents", "MacOS", "Example-App"),
		filepath.Join(artifact.Path, "Contents", "Resources", "server"),
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Mode()&0o111 == 0 {
			t.Errorf("%s mode = %v, want executable", path, info.Mode())
		}
	}
	runtimeData, err := os.ReadFile(filepath.Join(artifact.Path, "Contents", "Resources", manifest.RuntimeFileName))
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := manifest.RuntimeFromTOML(runtimeData)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.Server.Port != "8080" || runtime.WebView.URL != "http://127.0.0.1:8080/" {
		t.Errorf("runtime configuration = %#v", runtime)
	}
}

func TestPackageDebian(t *testing.T) {
	request := testRequest(t, TargetDebian)
	artifact, err := Package(request)
	if err != nil {
		t.Fatalf("Package() error = %v", err)
	}

	deb, err := os.ReadFile(artifact.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(deb, []byte("!<arch>\n")) {
		t.Fatal("package is not an ar archive")
	}
	entries, err := readAr(deb)
	if err != nil {
		t.Fatal(err)
	}
	controlArchive, ok := entries["control.tar.gz"]
	if !ok {
		t.Fatal("package does not contain control.tar.gz")
	}
	controlFiles, err := readTarGzip(controlArchive)
	if err != nil {
		t.Fatal(err)
	}
	control := string(controlFiles["control"])
	if !strings.Contains(control, "Depends: ") {
		t.Errorf("control does not declare Depends:\n%s", control)
	}
	if !strings.Contains(control, "libwebkit2gtk") {
		t.Errorf("control does not depend on the WebView runtime:\n%s", control)
	}

	data, ok := entries["data.tar.gz"]
	if !ok {
		t.Fatal("package does not contain data.tar.gz")
	}
	files, err := readTarGzip(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"usr/lib/com.example.app/bundle-core",
		"usr/lib/com.example.app/server",
		"usr/lib/com.example.app/" + manifest.RuntimeFileName,
		"usr/share/icons/hicolor/256x256/apps/com.example.app.png",
		"usr/share/applications/com.example.app.desktop",
	} {
		if _, ok := files[name]; !ok {
			t.Errorf("data.tar.gz does not contain %s", name)
		}
	}
}

func TestParseTarget(t *testing.T) {
	want := map[string]struct {
		platform     Target
		architecture string
	}{
		"darwin-amd64":  {TargetMacOS, "amd64"},
		"darwin-arm64":  {TargetMacOS, "arm64"},
		"linux-amd64":   {TargetDebian, "amd64"},
		"linux-arm64":   {TargetDebian, "arm64"},
		"windows-amd64": {TargetWindows, "amd64"},
		"windows-arm64": {TargetWindows, "arm64"},
	}
	if got := SupportedTargets(); len(got) != len(want) {
		t.Fatalf("SupportedTargets() = %v, want %d entries", got, len(want))
	}
	for triple, expected := range want {
		platform, architecture, ok := ParseTarget(triple)
		if !ok {
			t.Errorf("ParseTarget(%q) unsupported, want supported", triple)
			continue
		}
		if platform != expected.platform || architecture != expected.architecture {
			t.Errorf("ParseTarget(%q) = (%q, %q), want (%q, %q)", triple, platform, architecture, expected.platform, expected.architecture)
		}
	}
	if _, _, ok := ParseTarget("plan9-riscv64"); ok {
		t.Error("ParseTarget(plan9-riscv64) = supported, want unsupported")
	}
}

func TestICOBytes(t *testing.T) {
	data, err := icoBytes(assets{icon: testIcon(t)})
	if err != nil {
		t.Fatalf("icoBytes() error = %v", err)
	}
	if !bytes.HasPrefix(data, []byte{0, 0, 1, 0}) {
		t.Fatal("ICO output does not contain an ICO header")
	}
}

func testRequest(t *testing.T, target Target) Request {
	t.Helper()
	dir := t.TempDir()
	serverPath := filepath.Join(dir, "server")
	iconPath := filepath.Join(dir, "icon.png")
	if err := os.WriteFile(serverPath, []byte("server"), 0o755); err != nil {
		t.Fatal(err)
	}
	iconFile, err := os.Create(iconPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(iconFile, testIcon(t)); err != nil {
		iconFile.Close()
		t.Fatal(err)
	}
	if err := iconFile.Close(); err != nil {
		t.Fatal(err)
	}

	return Request{
		Target:       target,
		Architecture: "amd64",
		Core:         []byte("core"),
		Manifest: manifest.Manifest{
			App: manifest.App{
				Name:        "Example App",
				Version:     "1.2.3",
				ID:          "com.example.app",
				Description: "An example application",
				Maintainer:  "Example <example@example.com>",
				Icon:        iconPath,
			},
			Server:  manifest.Server{Binary: serverPath, RuntimeFlags: []string{"--port", "{port}"}, Port: "8080"},
			WebView: manifest.WebView{URL: "http://127.0.0.1:8080/"},
			Output:  manifest.Output{Path: filepath.Join(dir, "out")},
		},
	}
}

func testIcon(t *testing.T) image.Image {
	t.Helper()
	icon := image.NewNRGBA(image.Rect(0, 0, 512, 512))
	for y := range 512 {
		for x := range 512 {
			icon.Set(x, y, color.NRGBA{R: 0x20, G: 0x80, B: 0xe0, A: 0xff})
		}
	}
	return icon
}

func readAr(data []byte) (map[string][]byte, error) {
	if !bytes.HasPrefix(data, []byte("!<arch>\n")) {
		return nil, io.ErrUnexpectedEOF
	}
	entries := make(map[string][]byte)
	for offset := 8; offset < len(data); {
		if offset+60 > len(data) {
			return nil, io.ErrUnexpectedEOF
		}
		header := data[offset : offset+60]
		name := strings.TrimSuffix(strings.TrimSpace(string(header[:16])), "/")
		size, err := strconv.Atoi(strings.TrimSpace(string(header[48:58])))
		if err != nil {
			return nil, err
		}
		offset += 60
		if offset+size > len(data) {
			return nil, io.ErrUnexpectedEOF
		}
		entries[name] = data[offset : offset+size]
		offset += size
		if size%2 != 0 {
			offset++
		}
	}
	return entries, nil
}

func readTarGzip(data []byte) (map[string][]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	files := make(map[string][]byte)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return files, nil
		}
		if err != nil {
			return nil, err
		}
		contents, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, err
		}
		files[header.Name] = contents
	}
}
