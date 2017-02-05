// Package main implements a command line client for the Inbucket REST API
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

var host = flag.String("host", "localhost", "host/IP of Inbucket server")
var port = flag.Uint("port", 9000, "HTTP port of Inbucket server")

func main() {
	// Important top-level flags
	subcommands.ImportantFlag("host")
	subcommands.ImportantFlag("port")
	// Setup standard helpers
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	// Setup my commands
	subcommands.Register(&listCmd{}, "")
	subcommands.Register(&mboxCmd{}, "")
	// Parse and execute
	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}

func baseURL() string {
	return fmt.Sprintf("http://%s:%v", *host, *port)
}

func fatal(msg string, err error) subcommands.ExitStatus {
	fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	return subcommands.ExitFailure
}

func usage(msg string) subcommands.ExitStatus {
	fmt.Fprintln(os.Stderr, msg)
	return subcommands.ExitUsageError
}
