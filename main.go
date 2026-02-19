// Package main is the entry point for the speedtest application.
package main

import (
	"os"

	"github.com/nicholas-fedor/speedtest-go/cmd"
)

func main() {
	cmd.Init()

	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
