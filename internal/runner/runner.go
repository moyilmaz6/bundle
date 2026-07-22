package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/moyilmaz6/bundle/internal/manifest"
	webview "github.com/webview/webview_go"
)

const (
	startupTimeout          = 15 * time.Second
	readinessRequestTimeout = 500 * time.Millisecond
	pollInterval            = 100 * time.Millisecond
)

type Launch struct {
	Port string
	Args []string
	URL  string
}

func RunPackage(directory string) error {
	data, err := os.ReadFile(filepath.Join(directory, manifest.RuntimeFileName))
	if err != nil {
		return fmt.Errorf("read runtime configuration: %w", err)
	}
	runtime, err := manifest.RuntimeFromTOML(data)
	if err != nil {
		return fmt.Errorf("parse runtime configuration: %w", err)
	}
	return Run(directory, runtime)
}

func Run(directory string, runtime manifest.Runtime) error {
	launch, err := ResolveLaunch(runtime)
	if err != nil {
		return err
	}
	grace := runtime.Server.ShutdownGraceDuration()
	child, err := StartChild(directory, filepath.Join(directory, serverBinaryName()), launch.Args)
	if err != nil {
		return err
	}
	if err := waitStartup(child, launch.URL); err != nil {
		_ = child.Stop(grace)
		return err
	}

	view := webview.New(false)
	defer view.Destroy()
	view.SetTitle(runtime.Window.Title)
	view.SetSize(runtime.Window.Width, runtime.Window.Height, webview.HintNone)
	view.Navigate(launch.URL)

	windowDone := make(chan struct{})
	childExited := make(chan error, 1)
	go func() {
		err := child.Wait()
		select {
		case <-windowDone:
			return
		default:
			childExited <- err
			view.Terminate()
		}
	}()

	view.Run()
	close(windowDone)
	select {
	case err := <-childExited:
		if err == nil {
			return errors.New("server exited before the window closed")
		}
		return fmt.Errorf("server exited before the window closed: %w", err)
	default:
	}
	if err := child.Stop(grace); err != nil {
		return fmt.Errorf("stop server: %w", err)
	}
	return nil
}

func waitStartup(child *Child, url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()
	ready := make(chan error, 1)
	go func() { ready <- WaitReady(ctx, url, readinessClient(), pollInterval) }()
	select {
	case err := <-ready:
		return err
	case <-child.done:
		cancel()
		<-ready
		if err := child.Wait(); err != nil {
			return fmt.Errorf("server exited before it became ready: %w", err)
		}
		return errors.New("server exited before it became ready")
	}
}

func ResolveLaunch(runtime manifest.Runtime) (Launch, error) {
	if err := runtime.Validate(); err != nil {
		return Launch{}, fmt.Errorf("validate runtime configuration: %w", err)
	}
	port := runtime.Server.Port
	if port == "auto" {
		var err error
		port, err = availablePort()
		if err != nil {
			return Launch{}, fmt.Errorf("select a free port: %w", err)
		}
	}
	return Launch{
		Port: port,
		Args: substitutePort(runtime.Server.RuntimeFlags, port),
		URL:  strings.ReplaceAll(runtime.WebView.URL, "{port}", port),
	}, nil
}

func WaitReady(ctx context.Context, url string, client *http.Client, interval time.Duration) error {
	if err := requestReady(ctx, url, client); err == nil {
		return nil
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for web application at %s: %w", url, ctx.Err())
		case <-ticker.C:
			if err := requestReady(ctx, url, client); err == nil {
				return nil
			}
		}
	}
}

type Child struct {
	command *exec.Cmd
	done    chan struct{}
	mu      sync.Mutex
	err     error
}

func StartChild(directory, path string, args []string) (*Child, error) {
	command := exec.Command(path, args...)
	command.Dir = directory
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.SysProcAttr = childProcessAttributes()
	prepareChildLaunch()
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start server %s: %w", path, err)
	}
	child := &Child{command: command, done: make(chan struct{})}
	go func() {
		err := command.Wait()
		child.mu.Lock()
		child.err = err
		child.mu.Unlock()
		close(child.done)
	}()
	return child, nil
}

func (c *Child) Wait() error {
	<-c.done
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *Child) Stop(grace time.Duration) error {
	select {
	case <-c.done:
		return c.Wait()
	default:
	}
	return stopChild(c, grace)
}

func availablePort() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	return strconv.Itoa(port), nil
}

func substitutePort(values []string, port string) []string {
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = strings.ReplaceAll(value, "{port}", port)
	}
	return result
}

func readinessClient() *http.Client {
	return &http.Client{Timeout: readinessRequestTimeout}
}

func requestReady(ctx context.Context, url string, client *http.Client) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("unexpected HTTP status %d", response.StatusCode)
	}
	return nil
}
