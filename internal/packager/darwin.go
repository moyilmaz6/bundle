package packager

import (
	"fmt"
	"html"
	"os"
	"path/filepath"

	"github.com/jackmordaunt/icns/v3"
	"github.com/moyilmaz6/bundle/internal/bundl"
	"github.com/moyilmaz6/bundle/internal/manifest"
)

func packageMacOS(request Request, assets assets) (Artifact, error) {
	name := artifactName(request.Manifest.App.Name)
	appPath := filepath.Join(request.Manifest.Output.Path, name+".app")
	contentsPath := filepath.Join(appPath, "Contents")
	resourcesPath := filepath.Join(contentsPath, "Resources")

	if err := writeFile(filepath.Join(contentsPath, "Info.plist"), macOSInfoPlist(request), 0o644); err != nil {
		return Artifact{}, err
	}
	if err := writeFile(filepath.Join(contentsPath, "MacOS", name), assets.core, 0o755); err != nil {
		return Artifact{}, err
	}
	if err := writeFile(filepath.Join(resourcesPath, bundl.ServerName), assets.server, 0o755); err != nil {
		return Artifact{}, err
	}
	if err := writeFile(filepath.Join(resourcesPath, manifest.RuntimeFileName), assets.runtimeTOML, 0o644); err != nil {
		return Artifact{}, err
	}
	if err := writeICNS(filepath.Join(resourcesPath, "AppIcon.icns"), assets); err != nil {
		return Artifact{}, err
	}
	return Artifact{Path: appPath}, nil
}

func macOSInfoPlist(request Request) []byte {
	name := html.EscapeString(request.Manifest.App.Name)
	id := html.EscapeString(request.Manifest.App.ID)
	version := html.EscapeString(request.Manifest.App.Version)
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDisplayName</key><string>%s</string>
  <key>CFBundleExecutable</key><string>%s</string>
  <key>CFBundleIconFile</key><string>AppIcon</string>
  <key>CFBundleIdentifier</key><string>%s</string>
  <key>CFBundleName</key><string>%s</string>
  <key>CFBundleShortVersionString</key><string>%s</string>
  <key>CFBundleVersion</key><string>%s</string>
</dict>
</plist>
`, name, artifactName(request.Manifest.App.Name), id, name, version, version))
}

func writeICNS(path string, assets assets) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return icns.Encode(file, assets.icon)
}
