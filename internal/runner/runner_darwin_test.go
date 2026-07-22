//go:build darwin

package runner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/moyilmaz6/bundle/internal/manifest"
)

func TestResolveLaunch(t *testing.T) {
	tests := map[string]struct {
		runtime  manifest.Runtime
		wantPort string
	}{
		"fixed": {runtime: manifest.Runtime{Server: manifest.RuntimeServer{Port: "8080", RuntimeFlags: []string{"--port={port}"}}, WebView: manifest.WebView{URL: "http://127.0.0.1:{port}/"}}, wantPort: "8080"},
		"auto":  {runtime: manifest.Runtime{Server: manifest.RuntimeServer{Port: "auto", RuntimeFlags: []string{"--port", "{port}"}}, WebView: manifest.WebView{URL: "http://127.0.0.1:{port}/"}}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			launch, err := ResolveLaunch(tc.runtime)
			if err != nil {
				t.Fatalf("ResolveLaunch() error = %v", err)
			}
			if tc.wantPort != "" && launch.Port != tc.wantPort {
				t.Fatalf("port = %q, want %q", launch.Port, tc.wantPort)
			}
			if _, err := strconv.Atoi(launch.Port); err != nil {
				t.Fatalf("port = %q, not numeric", launch.Port)
			}
			if launch.URL != "http://127.0.0.1:"+launch.Port+"/" {
				t.Fatalf("URL = %q", launch.URL)
			}
		})
	}
}

func TestWaitReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)
	if err := WaitReady(ctx, server.URL, readinessClient(), time.Millisecond); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}
}

func TestWaitReadyTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	t.Cleanup(cancel)
	if err := WaitReady(ctx, "http://127.0.0.1:1/", readinessClient(), time.Millisecond); err == nil {
		t.Fatal("WaitReady() error = nil, want timeout")
	}
}

func TestChildStop(t *testing.T) {
	child, err := StartChild(t.TempDir(), "/bin/sh", []string{"-c", "trap 'exit 0' TERM; while :; do sleep 1; done"})
	if err != nil {
		t.Fatalf("StartChild() error = %v", err)
	}
	if err := child.Stop(5 * time.Second); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}
