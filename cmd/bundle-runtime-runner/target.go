package main

import "runtime"

func runtimeTarget() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}
