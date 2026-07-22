package packager

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/moyilmaz6/bundle/internal/bundl"
	"github.com/sergeymakinen/go-ico"
	"github.com/tc-hib/winres"
	"golang.org/x/image/draw"
)

func packageWindows(request Request, assets assets) (Artifact, error) {
	core, err := patchWindowsCore(assets.core, assets)
	if err != nil {
		return Artifact{}, err
	}
	name := artifactName(request.Manifest.App.Name)
	outputPath := filepath.Join(request.Manifest.Output.Path, fmt.Sprintf("%s-windows-%s.zip", name, request.Architecture))
	if err := os.MkdirAll(request.Manifest.Output.Path, 0o755); err != nil {
		return Artifact{}, err
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	defer archive.Close()
	root := name + "/"
	if err := writeZIPFile(archive, root+name+".exe", core, 0o755); err != nil {
		return Artifact{}, err
	}
	if err := writeZIPFile(archive, root+bundl.ServerNameWindows, assets.server, 0o755); err != nil {
		return Artifact{}, err
	}
	if err := writeZIPFile(archive, root+"bundle-runtime.toml", assets.runtimeTOML, 0o644); err != nil {
		return Artifact{}, err
	}
	icoData, err := icoBytes(assets)
	if err != nil {
		return Artifact{}, err
	}
	if err := writeZIPFile(archive, root+"app.ico", icoData, 0o644); err != nil {
		return Artifact{}, err
	}
	return Artifact{Path: outputPath}, nil
}

func patchWindowsCore(core []byte, assets assets) ([]byte, error) {
	file := bytes.NewReader(core)
	resources, err := winres.LoadFromEXE(file)
	if err != nil {
		return nil, fmt.Errorf("load Windows core resources: %w", err)
	}
	icon, err := winres.NewIconFromResizedImage(assets.icon, nil)
	if err != nil {
		return nil, err
	}
	if err := resources.SetIcon(winres.ID(1), icon); err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	var patched bytes.Buffer
	if err := resources.WriteToEXE(&patched, file, winres.ForceCheckSum()); err != nil {
		return nil, fmt.Errorf("embed Windows icon: %w", err)
	}
	return patched.Bytes(), nil
}

func icoBytes(assets assets) ([]byte, error) {
	var output bytes.Buffer
	images := make([]image.Image, 0, len(winres.DefaultIconSizes))
	for _, size := range winres.DefaultIconSizes {
		images = append(images, resizeIcon(assets.icon, size))
	}
	if err := ico.EncodeAll(&output, images); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func resizeIcon(source image.Image, size int) image.Image {
	result := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.CatmullRom.Scale(result, result.Bounds(), source, source.Bounds(), draw.Over, nil)
	return result
}

func writeZIPFile(archive *zip.Writer, name string, data []byte, mode os.FileMode) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(mode)
	file, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	return err
}
