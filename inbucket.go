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
	golog "log"
	"os"
	"time"
)

var help = flag.Bool("help", false, "Displays this help")
var pidfile = flag.String("pidfile", "none", "Write our PID into the specified file")
var logfile = flag.String("logfile", "stderr", "Write out log into the specified file")

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

	// Configure logging, close std* fds
	if *logfile != "stderr" {
		// stderr is the go logging default
		if *logfile == "stdout" {
			// set to stdout
			golog.SetOutput(os.Stdout)
		} else {
			// use specificed log file
			logf, err := os.OpenFile(*logfile, os.O_WRONLY | os.O_APPEND | os.O_CREATE, 0666)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create %v: %v\n", *logfile, err)
				os.Exit(1)
			}
			defer logf.Close()
			golog.SetOutput(logf)

			// close std* streams
			os.Stdout.Close()
			os.Stderr.Close()
			os.Stdin.Close()
			os.Stdout = logf
			os.Stderr = logf
		}
	}

	level, _ := config.Config.String("logging", "level")
	log.SetLogLevel(level)

	// Write pidfile if requested
	// TODO: Probably supposed to remove pidfile during shutdown
	if *pidfile != "none" {
		pidf, err := os.Create(*pidfile)
		if err != nil {
			log.Error("Failed to create %v: %v", *pidfile, err)
			os.Exit(1)
		}
		defer pidf.Close()
		fmt.Fprintf(pidf, "%v\n", os.Getpid())
	}

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
