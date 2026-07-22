package packager

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/moyilmaz6/bundle/internal/manifest"
)

type tarFile struct {
	name string
	data []byte
	mode int64
}

func packageDebian(request Request, assets assets) (Artifact, error) {
	packageName := debianPackageName(request.Manifest.App.ID)
	architecture := request.Architecture
	if architecture == "" {
		return Artifact{}, fmt.Errorf("Debian package architecture is required")
	}
	if request.Manifest.App.Maintainer == "" {
		return Artifact{}, fmt.Errorf("app.maintainer is required for Debian packaging")
	}
	if request.Manifest.App.Description == "" {
		return Artifact{}, fmt.Errorf("app.description is required for Debian packaging")
	}

	control, err := tarGzip([]tarFile{{name: "control", data: debianControl(request, packageName), mode: 0o644}})
	if err != nil {
		return Artifact{}, err
	}
	data, err := tarGzip([]tarFile{
		{name: "usr/lib/" + packageName + "/bundle-core", data: assets.core, mode: 0o755},
		{name: "usr/lib/" + packageName + "/server", data: assets.server, mode: 0o755},
		{name: "usr/lib/" + packageName + "/" + manifest.RuntimeFileName, data: assets.runtimeTOML, mode: 0o644},
		{name: "usr/share/icons/hicolor/256x256/apps/" + packageName + ".png", data: assets.iconPNG, mode: 0o644},
		{name: "usr/share/applications/" + packageName + ".desktop", data: debianDesktopEntry(request, packageName), mode: 0o644},
	})
	if err != nil {
		return Artifact{}, err
	}

	outputPath := filepath.Join(request.Manifest.Output.Path, fmt.Sprintf("%s_%s_%s.deb", packageName, request.Manifest.App.Version, architecture))
	var archive bytes.Buffer
	archive.WriteString("!<arch>\n")
	if err := writeArFile(&archive, "debian-binary", []byte("2.0\n")); err != nil {
		return Artifact{}, err
	}
	if err := writeArFile(&archive, "control.tar.gz", control); err != nil {
		return Artifact{}, err
	}
	if err := writeArFile(&archive, "data.tar.gz", data); err != nil {
		return Artifact{}, err
	}
	if err := writeFile(outputPath, archive.Bytes(), 0o644); err != nil {
		return Artifact{}, err
	}
	return Artifact{Path: outputPath}, nil
}

const debianCoreDepends = "libgtk-3-0, libwebkit2gtk-4.0-37"

func debianControl(request Request, packageName string) []byte {
	return []byte(fmt.Sprintf("Package: %s\nVersion: %s\nArchitecture: %s\nDepends: %s\nMaintainer: %s\nDescription: %s\n", packageName, request.Manifest.App.Version, request.Architecture, debianCoreDepends, request.Manifest.App.Maintainer, debianDescription(request.Manifest.App.Description)))
}

func debianDesktopEntry(request Request, packageName string) []byte {
	return []byte(fmt.Sprintf("[Desktop Entry]\nType=Application\nName=%s\nComment=%s\nExec=/usr/lib/%s/bundle-core\nIcon=%s\nTerminal=false\nCategories=Utility;\n", request.Manifest.App.Name, request.Manifest.App.Description, packageName, packageName))
}

func tarGzip(files []tarFile) ([]byte, error) {
	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, file := range files {
		header := &tar.Header{Name: file.name, Mode: file.mode, Size: int64(len(file.data)), ModTime: time.Unix(0, 0)}
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, err
		}
		if _, err := tarWriter.Write(file.data); err != nil {
			return nil, err
		}
	}
	if err := tarWriter.Close(); err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func writeArFile(archive *bytes.Buffer, name string, data []byte) error {
	if len(name) > 15 {
		return fmt.Errorf("ar file name %q is too long", name)
	}
	header := fmt.Sprintf("%-16s%-12d%-6d%-6d%-8o%-10d`\n", name+"/", 0, 0, 0, 0o100644, len(data))
	if len(header) != 60 {
		return fmt.Errorf("invalid ar header length %d", len(header))
	}
	archive.WriteString(header)
	archive.Write(data)
	if len(data)%2 != 0 {
		archive.WriteByte('\n')
	}
	return nil
}

func debianDescription(value string) string {
	return strings.ReplaceAll(value, "\n", "\n ")
}
