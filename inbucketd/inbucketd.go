/*
	This is the inbucket daemon launcher
*/
package main

import (
	"flag"
	"fmt"
	"github.com/jhillyerd/inbucket"
	"github.com/jhillyerd/inbucket/smtpd"
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
	configError(err)

	// Startup SMTP server
	domain, err := inbucket.Config.String("smtp", "domain")
	configError(err)
	port, err := inbucket.Config.Int("smtp", "ip4.port")
	configError(err)
	server := smtpd.New(domain, port)
	go server.Start()
}

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage of inbucketd [options] <conf file>:")
		flag.PrintDefaults()
	}
}

func configError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config file: %v\n", err)
		os.Exit(1)
	}
}
