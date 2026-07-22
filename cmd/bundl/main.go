package main

import (
	"os"

	"github.com/moyilmaz6/bundle/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
