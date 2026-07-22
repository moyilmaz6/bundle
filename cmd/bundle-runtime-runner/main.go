package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moyilmaz6/bundle/internal/bundl"
	"github.com/moyilmaz6/bundle/internal/runner"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: bundle-runtime-runner <package.bundl>")
		os.Exit(2)
	}
	directory, err := filepath.Abs(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "bundle runtime:", err)
		os.Exit(1)
	}
	runtime, err := bundl.Open(directory, runtimeTarget())
	if err != nil {
		fmt.Fprintln(os.Stderr, "bundle runtime:", err)
		os.Exit(1)
	}
	if err := runner.Run(directory, runtime); err != nil {
		fmt.Fprintln(os.Stderr, "bundle runtime:", err)
		os.Exit(1)
	}
}
