//go:build darwin || linux

package runner

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"github.com/moyilmaz6/bundle/internal/bundl"
)

func serverBinaryName() string { return bundl.ServerName }

func prepareChildLaunch() {}

func childProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func stopChild(c *Child, grace time.Duration) error {
	if err := syscall.Kill(-c.command.Process.Pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("send SIGTERM: %w", err)
	}
	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-c.done:
		return expectedShutdown(c.Wait())
	case <-timer.C:
		if err := syscall.Kill(-c.command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("send SIGKILL: %w", err)
		}
		return expectedShutdown(c.Wait())
	}
}

func expectedShutdown(err error) error {
	if err == nil {
		return nil
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		return err
	}
	status, ok := exitError.Sys().(syscall.WaitStatus)
	if ok && status.Signaled() && (status.Signal() == syscall.SIGTERM || status.Signal() == syscall.SIGKILL) {
		return nil
	}
	return err
}
