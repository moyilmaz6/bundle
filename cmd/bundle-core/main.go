package main

import (
	"fmt"
	"os"

	"github.com/moyilmaz6/bundle/internal/core"
)

func main() {
	if err := core.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "bundle core:", err)
		os.Exit(1)
	}
}
