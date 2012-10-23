/*
	This is the inbucket daemon launcher
*/
package main

import (
	"expvar"
	"flag"
	"fmt"
	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/log"
	"github.com/jhillyerd/inbucket/smtpd"
	"github.com/jhillyerd/inbucket/web"
	"os"
	"time"
)

var help = flag.Bool("help", false, "Displays this help")

var startTime = time.Now()

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

	// Configure logging
	level, _ := config.Config.String("logging", "level")
	log.SetLogLevel(level)

	// Startup SMTP server
	server := smtpd.New()
	go server.Start()

	web.Start()
}

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage of inbucket [options] <conf file>:")
		flag.PrintDefaults()
	}

	expvar.Publish("uptime", expvar.Func(uptime))
}

func uptime() interface{} {
	return time.Since(startTime) / time.Second
}
