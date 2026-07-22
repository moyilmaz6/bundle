package manifest

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	FileName        = "bundle.toml"
	RuntimeFileName = "bundle-runtime.toml"
)

const (
	DefaultWindowWidth  = 1200
	DefaultWindowHeight = 800
)

const DefaultShutdownGrace = "5s"

type Manifest struct {
	App     App     `toml:"app"`
	Server  Server  `toml:"server"`
	WebView WebView `toml:"webview"`
	Window  Window  `toml:"window"`
	Output  Output  `toml:"output"`
}

type App struct {
	Name        string `toml:"name"`
	Version     string `toml:"version"`
	ID          string `toml:"id"`
	Description string `toml:"description"`
	Maintainer  string `toml:"maintainer"`
	Icon        string `toml:"icon"`
}

type Server struct {
	Binary        string   `toml:"binary"`
	RuntimeFlags  []string `toml:"runtime_flags"`
	Port          string   `toml:"port"`
	ShutdownGrace string   `toml:"shutdown_grace"`
}

type WebView struct {
	URL string `toml:"url"`
}

type Window struct {
	Title  string `toml:"title"`
	Width  int    `toml:"width"`
	Height int    `toml:"height"`
}

type Output struct {
	Path string `toml:"path"`
}

type Runtime struct {
	Server  RuntimeServer `toml:"server"`
	WebView WebView       `toml:"webview"`
	Window  Window        `toml:"window"`
}

type RuntimeServer struct {
	RuntimeFlags  []string `toml:"runtime_flags"`
	Port          string   `toml:"port"`
	ShutdownGrace string   `toml:"shutdown_grace"`
}

func (r RuntimeServer) ShutdownGraceDuration() time.Duration {
	grace, err := time.ParseDuration(r.ShutdownGrace)
	if err != nil || grace < 0 {
		grace, _ = time.ParseDuration(DefaultShutdownGrace)
	}
	return grace
}

func ManifestFromTOML(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := toml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest.WithDefaults(), nil
}

func RuntimeFromTOML(data []byte) (Runtime, error) {
	var runtime Runtime
	if err := toml.Unmarshal(data, &runtime); err != nil {
		return Runtime{}, err
	}
	return runtime.WithDefaults(), nil
}

func (m Manifest) ToTOML() ([]byte, error) {
	return toml.Marshal(m.WithDefaults())
}

func (r Runtime) ToTOML() ([]byte, error) {
	return toml.Marshal(r.WithDefaults())
}

func (m Manifest) WithDefaults() Manifest {
	if m.Window.Title == "" {
		m.Window.Title = m.App.Name
	}
	if m.Window.Width == 0 {
		m.Window.Width = DefaultWindowWidth
	}
	if m.Window.Height == 0 {
		m.Window.Height = DefaultWindowHeight
	}
	if m.Server.ShutdownGrace == "" {
		m.Server.ShutdownGrace = DefaultShutdownGrace
	}
	return m
}

func (r Runtime) WithDefaults() Runtime {
	if r.Window.Width == 0 {
		r.Window.Width = DefaultWindowWidth
	}
	if r.Window.Height == 0 {
		r.Window.Height = DefaultWindowHeight
	}
	if r.Server.ShutdownGrace == "" {
		r.Server.ShutdownGrace = DefaultShutdownGrace
	}
	return r
}

func (m Manifest) ResolvePaths(basePath string) Manifest {
	m.App.Icon = resolvePath(basePath, m.App.Icon)
	m.Server.Binary = resolvePath(basePath, m.Server.Binary)
	m.Output.Path = resolvePath(basePath, m.Output.Path)
	return m
}

func (m Manifest) Runtime() Runtime {
	m = m.WithDefaults()
	return Runtime{
		Server:  RuntimeServer{RuntimeFlags: m.Server.RuntimeFlags, Port: m.Server.Port, ShutdownGrace: m.Server.ShutdownGrace},
		WebView: m.WebView,
		Window:  m.Window,
	}
}

func (m Manifest) Validate() error {
	m = m.WithDefaults()
	if m.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}
	if m.App.Version == "" {
		return fmt.Errorf("app.version is required")
	}
	if m.App.ID == "" {
		return fmt.Errorf("app.id is required")
	}
	if m.App.Icon == "" {
		return fmt.Errorf("app.icon is required")
	}
	if m.Server.Binary == "" {
		return fmt.Errorf("server.binary is required")
	}
	if m.Output.Path == "" {
		return fmt.Errorf("output.path is required")
	}
	return m.Runtime().Validate()
}

func (r Runtime) Validate() error {
	r = r.WithDefaults()
	if r.Server.Port == "" {
		return fmt.Errorf("server.port is required")
	}
	if r.WebView.URL == "" {
		return fmt.Errorf("webview.url is required")
	}
	if r.Window.Width <= 0 || r.Window.Height <= 0 {
		return fmt.Errorf("window width and height must be positive")
	}
	if grace, err := time.ParseDuration(r.Server.ShutdownGrace); err != nil || grace < 0 {
		return fmt.Errorf("server.shutdown_grace must be a non-negative duration such as %q", DefaultShutdownGrace)
	}
	if r.Server.Port == "auto" {
		if !containsPortPlaceholder(r.Server.RuntimeFlags) {
			return fmt.Errorf("server.runtime_flags must contain {port} when server.port is auto")
		}
		if !strings.Contains(r.WebView.URL, "{port}") {
			return fmt.Errorf("webview.url must contain {port} when server.port is auto")
		}
		return nil
	}
	port, err := strconv.Atoi(r.Server.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("server.port must be auto or a value from 1 through 65535")
	}
	return nil
}

func resolvePath(basePath, value string) string {
	if value == "" || filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(basePath, value)
}

func containsPortPlaceholder(flags []string) bool {
	for _, flag := range flags {
		if strings.Contains(flag, "{port}") {
			return true
		}
	}
	return false
}

func TemplateTOML() []byte {
	return []byte(`[app]
# Human-readable application name.
name = "My App"

# Application release version.
version = "0.1.0"

# Stable bundle/package identifier.
id = "com.example.myapp"

# Short application description.
description = "My Bundle application"

# Package maintainer, required by Debian packages.
maintainer = "Your Name <you@example.com>"

# Path to the application icon source PNG file.
icon = "./assets/icon.png"

[server]
# Path to the prebuilt server binary that Bundle packages.
binary = "./my-server"

# Arguments passed when Bundle starts the server. Use {port} for the selected port.
runtime_flags = ["--port", "{port}"]

# "auto" finds a free local port. Use a number in quotes to keep a fixed port.
port = "auto"

# When the window closes, the server is asked to stop (SIGTERM; CTRL+Break on
# Windows) and given this long to exit before it is force-killed. "0" stops it
# immediately. Your server needs no special code: handle the OS signal as usual.
shutdown_grace = "5s"

[webview]
# URL opened by the WebView after the server responds. Use {port} with auto ports.
url = "http://127.0.0.1:{port}/"

[window]
# Window title. Leave blank to use app.name.
title = ""

# Initial window size in pixels.
width = 1200
height = 800

[output]
# Directory where Bundle writes the final application or .bundl package.
path = "./out"
`)
}
