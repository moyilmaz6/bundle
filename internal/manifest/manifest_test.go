package manifest

import (
	"bytes"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestManifestTOMLRoundTrip(t *testing.T) {
	want := Manifest{
		App:     App{Name: "Example App", Version: "1.2.3", ID: "com.example.app", Description: "An example application", Maintainer: "Example <example@example.com>", Icon: "./assets/icon.png"},
		Server:  Server{Binary: "./example-server", RuntimeFlags: []string{"--port", "{port}"}, Port: "auto", ShutdownGrace: "3s"},
		WebView: WebView{URL: "http://127.0.0.1:{port}/dashboard"},
		Window:  Window{Title: "Example App", Width: 900, Height: 700},
		Output:  Output{Path: "./dist"},
	}

	data, err := want.ToTOML()
	if err != nil {
		t.Fatalf("ToTOML() error = %v", err)
	}
	got, err := ManifestFromTOML(data)
	if err != nil {
		t.Fatalf("ManifestFromTOML() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ManifestFromTOML(ToTOML()) = %#v, want %#v", got, want)
	}
}

func TestManifestDefaultsAndPathResolution(t *testing.T) {
	manifest := Manifest{
		App:    App{Name: "Example", Icon: "assets/icon.png"},
		Server: Server{Binary: "bin/server"},
		Output: Output{Path: "out"},
	}
	got := manifest.WithDefaults().ResolvePaths("/workspace/app")
	if got.Window.Title != "Example" || got.Window.Width != DefaultWindowWidth || got.Window.Height != DefaultWindowHeight {
		t.Fatalf("window defaults = %#v", got.Window)
	}
	if got.App.Icon != filepath.Join("/workspace/app", "assets/icon.png") || got.Server.Binary != filepath.Join("/workspace/app", "bin/server") || got.Output.Path != filepath.Join("/workspace/app", "out") {
		t.Fatalf("resolved paths = %#v", got)
	}
}

func TestRuntimeValidate(t *testing.T) {
	tests := map[string]struct {
		runtime Runtime
		wantErr bool
	}{
		"auto_port":                     {runtime: Runtime{Server: RuntimeServer{Port: "auto", RuntimeFlags: []string{"--port={port}"}}, WebView: WebView{URL: "http://127.0.0.1:{port}/"}}},
		"fixed_port":                    {runtime: Runtime{Server: RuntimeServer{Port: "8080"}, WebView: WebView{URL: "http://127.0.0.1:8080/"}}},
		"auto_without_flag_placeholder": {runtime: Runtime{Server: RuntimeServer{Port: "auto"}, WebView: WebView{URL: "http://127.0.0.1:{port}/"}}, wantErr: true},
		"invalid_port":                  {runtime: Runtime{Server: RuntimeServer{Port: "70000"}, WebView: WebView{URL: "http://127.0.0.1:8080/"}}, wantErr: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.runtime.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestShutdownGrace(t *testing.T) {

	got := Manifest{
		App:    App{Name: "Example", Icon: "assets/icon.png"},
		Server: Server{Binary: "bin/server"},
		Output: Output{Path: "out"},
	}.WithDefaults()
	if got.Server.ShutdownGrace != DefaultShutdownGrace {
		t.Fatalf("default shutdown_grace = %q, want %q", got.Server.ShutdownGrace, DefaultShutdownGrace)
	}
	if grace := got.Runtime().Server.ShutdownGraceDuration(); grace != 5*time.Second {
		t.Fatalf("ShutdownGraceDuration() = %v, want 5s", grace)
	}

	if grace := (RuntimeServer{ShutdownGrace: "0"}).ShutdownGraceDuration(); grace != 0 {
		t.Fatalf("ShutdownGraceDuration(\"0\") = %v, want 0", grace)
	}

	for _, value := range []string{"soon", "-1s"} {
		runtime := Runtime{Server: RuntimeServer{Port: "8080", ShutdownGrace: value}, WebView: WebView{URL: "http://127.0.0.1:8080/"}}
		if err := runtime.Validate(); err == nil {
			t.Errorf("Validate() with shutdown_grace %q = nil, want error", value)
		}
	}
}

func TestTemplateTOML(t *testing.T) {
	data := TemplateTOML()
	if !bytes.Contains(data, []byte("# Window title. Leave blank to use app.name.")) {
		t.Fatal("TemplateTOML() does not contain window field comments")
	}
	got, err := ManifestFromTOML(data)
	if err != nil {
		t.Fatalf("ManifestFromTOML(TemplateTOML()) error = %v", err)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("template validation error = %v", err)
	}
	if got.Server.Port != "auto" || got.Window.Title != "My App" || got.Window.Width != 1200 || got.Window.Height != 800 {
		t.Fatalf("template defaults = %#v", got)
	}
}
