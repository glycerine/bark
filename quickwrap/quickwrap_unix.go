//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"os"

	"github.com/glycerine/bark"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s <command> [args...]\n", os.Args[0])
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	// On non-Windows platforms, use standard functions
	w, err := bark.StartAndWatch(cmd, args...)
	if err != nil {
		fmt.Printf("Error starting process: %v\n", err)
		os.Exit(1)
	}

	// Wait for the process to finish
	<-w.Done
	os.Exit(w.ExitCode)
}
