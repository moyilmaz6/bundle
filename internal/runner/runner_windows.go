//go:build windows

package runner

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/moyilmaz6/bundle/internal/bundl"
)

const terminationWait = 5 * time.Second

const (
	ctrlBreakEvent = 1
	swHide         = 0
)

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	user32                       = syscall.NewLazyDLL("user32.dll")
	procAllocConsole             = kernel32.NewProc("AllocConsole")
	procGetConsoleWindow         = kernel32.NewProc("GetConsoleWindow")
	procGenerateConsoleCtrlEvent = kernel32.NewProc("GenerateConsoleCtrlEvent")
	procShowWindow               = user32.NewProc("ShowWindow")
	consoleOnce                  sync.Once
)

func serverBinaryName() string { return bundl.ServerNameWindows }

func childProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func prepareChildLaunch() {
	consoleOnce.Do(func() {
		if hwnd, _, _ := procGetConsoleWindow.Call(); hwnd != 0 {
			return
		}
		procAllocConsole.Call()
		if hwnd, _, _ := procGetConsoleWindow.Call(); hwnd != 0 {
			procShowWindow.Call(hwnd, uintptr(swHide))
		}
	})
}

func stopChild(c *Child, grace time.Duration) error {
	sendCtrlBreak(c.command.Process.Pid)
	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-c.done:
		return nil
	case <-timer.C:
	}
	if err := exec.Command("taskkill", "/PID", strconv.Itoa(c.command.Process.Pid), "/T", "/F").Run(); err != nil {
		return fmt.Errorf("terminate server process tree: %w", err)
	}
	select {
	case <-c.done:
		return nil
	case <-time.After(terminationWait):
		return fmt.Errorf("wait for server shutdown: %w", context.DeadlineExceeded)
	}
}

func sendCtrlBreak(pid int) {
	procGenerateConsoleCtrlEvent.Call(uintptr(ctrlBreakEvent), uintptr(pid))
}
