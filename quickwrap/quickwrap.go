package main

import (
	"fmt"
	"os"

	ec "github.com/glycerine/bark"
)

func main() {
	if len(os.Args) <= 1 {
		fmt.Fprintf(os.Stderr, "no command provided. use: quickwrap command args...\n")
		os.Exit(1)
	}
	addme := 45
	fmt.Printf("os.Args = %#v\n", os.Args)
	exitCode, _ := ec.OneshotAndWait(os.Args[1], 0, os.Args[2:]...)
	fmt.Printf("\n ---> exitCode was 0x%x (%v decimal), adding %v<<8 and returning %v\n",
		exitCode, exitCode, addme, (exitCode+(addme<<8))>>8)
	os.Exit((addme<<8 + exitCode) >> 8)
}
