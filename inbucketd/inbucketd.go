/*
	This is the inbucket daemon launcher
*/
package main

import (
	"flag"
	"fmt"
	"github.com/jhillyerd/inbucket"
	"github.com/jhillyerd/inbucket/smtpd"
	"log"
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
	err := inbucket.LoadConfig(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse config: %v\n", err)
		os.Exit(1)
	}

	log.Println("Logger test")
	inbucket.Trace("trace test")
	inbucket.Info("info test")
	inbucket.Warn("warn test")
	inbucket.Error("error test")

	// Startup SMTP server
	server := smtpd.New()
	server.Start()
}

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage of inbucketd [options] <conf file>:")
		flag.PrintDefaults()
	}
}
