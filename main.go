// Command toggl-cli is a command line interface for Toggl Track.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/ville6000/toggl-cli/cmd"
)

func main() {
	// Cancel in-flight API requests on Ctrl-C.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := cmd.Execute(ctx)
	stop()

	if err != nil {
		os.Exit(1)
	}
}
