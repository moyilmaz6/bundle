package packager

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/moyilmaz6/bundle/internal/manifest"
)

type Target string

const (
	TargetMacOS   Target = "darwin"
	TargetWindows Target = "windows"
	TargetDebian  Target = "debian"
)

type targetInfo struct {
	target       Target
	architecture string
}

var targetsByTriple = map[string]targetInfo{
	"darwin-amd64":  {TargetMacOS, "amd64"},
	"darwin-arm64":  {TargetMacOS, "arm64"},
	"linux-amd64":   {TargetDebian, "amd64"},
	"linux-arm64":   {TargetDebian, "arm64"},
	"windows-amd64": {TargetWindows, "amd64"},
	"windows-arm64": {TargetWindows, "arm64"},
}

func ParseTarget(triple string) (Target, string, bool) {
	info, ok := targetsByTriple[triple]
	return info.target, info.architecture, ok
}

func IsSupportedTarget(triple string) bool {
	_, ok := targetsByTriple[triple]
	return ok
}

func SupportedTargets() []string {
	triples := make([]string, 0, len(targetsByTriple))
	for triple := range targetsByTriple {
		triples = append(triples, triple)
	}
	sort.Strings(triples)
	return triples
}

type Request struct {
	Manifest     manifest.Manifest
	Target       Target
	Architecture string
	Core         []byte
}

type Artifact struct {
	Path string
}

type assets struct {
	core        []byte
	server      []byte
	runtimeTOML []byte
	icon        image.Image
	iconPNG     []byte
}

func Package(request Request) (Artifact, error) {
	if request.Manifest.Output.Path == "" {
		return Artifact{}, fmt.Errorf("output.path is required")
	}
	if len(request.Core) == 0 {
		return Artifact{}, fmt.Errorf("core is required")
	}
	if request.Manifest.Server.Binary == "" {
		return Artifact{}, fmt.Errorf("server.binary is required")
	}
	if request.Manifest.App.Icon == "" {
		return Artifact{}, fmt.Errorf("app.icon is required")
	}

	assets, err := loadAssets(request)
	if err != nil {
		return Artifact{}, err
	}

	switch request.Target {
	case TargetMacOS:
		return packageMacOS(request, assets)
	case TargetWindows:
		return packageWindows(request, assets)
	case TargetDebian:
		return packageDebian(request, assets)
	default:
		return Artifact{}, fmt.Errorf("unsupported package target %q", request.Target)
	}
}

func loadAssets(request Request) (assets, error) {
	server, err := os.ReadFile(request.Manifest.Server.Binary)
	if err != nil {
		return assets{}, fmt.Errorf("read server binary: %w", err)
	}
	iconPNG, err := os.ReadFile(request.Manifest.App.Icon)
	if err != nil {
		return assets{}, fmt.Errorf("read app icon: %w", err)
	}
	icon, err := png.Decode(bytes.NewReader(iconPNG))
	if err != nil {
		return assets{}, fmt.Errorf("decode app icon as PNG: %w", err)
	}
	bounds := icon.Bounds()
	if bounds.Dx() != bounds.Dy() {
		return assets{}, fmt.Errorf("app icon must be square")
	}
	runtimeTOML, err := request.Manifest.Runtime().ToTOML()
	if err != nil {
		return assets{}, fmt.Errorf("encode runtime configuration: %w", err)
	}
	return assets{core: request.Core, server: server, runtimeTOML: runtimeTOML, icon: icon, iconPNG: iconPNG}, nil
}

func artifactName(name string) string {
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, name)
	name = strings.Trim(name, "-.")
	if name == "" {
		return "bundle-app"
	}
	return name
}

func debianPackageName(id string) string {
	id = strings.ToLower(id)
	id = strings.Map(func(r rune) rune {
		if unicode.IsLower(r) || unicode.IsDigit(r) || r == '+' || r == '-' || r == '.' {
			return r
		}
		return '-'
	}, id)
	id = strings.Trim(id, "-.")
	if id == "" {
		return "bundle-app"
	}
	return id
}

func writeFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, mode)
}
