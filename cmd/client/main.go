// Package main implements a command line client for the Inbucket REST API
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"

	"github.com/google/subcommands"
)

var host = flag.String("host", "localhost", "host/IP of Inbucket server")
var port = flag.Uint("port", 9000, "HTTP port of Inbucket server")

// Allow subcommands to accept regular expressions as flags
type regexFlag struct {
	*regexp.Regexp
}

func (r *regexFlag) Defined() bool {
	return r.Regexp != nil
}

func (r *regexFlag) Set(pattern string) error {
	if pattern == "" {
		r.Regexp = nil
		return nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	r.Regexp = re
	return nil
}

func (r *regexFlag) String() string {
	if r.Regexp == nil {
		return ""
	}
	return r.Regexp.String()
}

// regexFlag must implement flag.Value
var _ flag.Value = &regexFlag{}

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
	subcommands.Register(&matchCmd{}, "")
	subcommands.Register(&mboxCmd{}, "")

	// Parse and execute
	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}

func baseURL() string {
	return "http://%s" + net.JoinHostPort(*host, strconv.FormatUint(uint64(*port), 10))
}

func fatal(msg string, err error) subcommands.ExitStatus {
	fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
	return subcommands.ExitFailure
}

func usage(msg string) subcommands.ExitStatus {
	fmt.Fprintln(os.Stderr, msg)
	return subcommands.ExitUsageError
}
