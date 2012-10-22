/*
	This is the inbucket daemon launcher
*/
package main

import (
	"flag"
	"fmt"
	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/smtpd"
	"github.com/jhillyerd/inbucket/web"
	"os"
)

var help = flag.Bool("help", false, "Displays this help")

func main() {
	flag.Parse()
	if *help {
		flag.Usage()
		return
	}

	// Load & Parse config
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	err := config.LoadConfig(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse config: %v\n", err)
		os.Exit(1)
	}

	// Startup SMTP server
	server := smtpd.New()
	go server.Start()

	web.Start()
}

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage of inbucketd [options] <conf file>:")
		flag.PrintDefaults()
	}
}
