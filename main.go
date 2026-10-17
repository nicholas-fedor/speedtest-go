// Package main is the entry point for the speedtest application.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/nicholas-fedor/speedtest-go/v2/internal/app"
)

// main runs the command the arguments name and exits with its code.
// The first interrupt cancels the command's context.
func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
	)

	// The first interrupt cancels the run.
	// Restoring the default handling lets a second one end the process at once
	// when the run does not stop.
	go func() {
		<-ctx.Done()
		stop()
	}()

	code := app.Run(ctx)

	stop()

	os.Exit(code)
}
