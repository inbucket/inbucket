// main is the inbucket daemon launcher
package main

import (
	"expvar"
	"flag"
	"fmt"
	golog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/httpd"
	"github.com/jhillyerd/inbucket/log"
	"github.com/jhillyerd/inbucket/pop3d"
	"github.com/jhillyerd/inbucket/smtpd"
	"github.com/jhillyerd/inbucket/webui"
)

var (
	// VERSION contains the build version number, populated during linking by goxc
	VERSION = "1.1.0.snapshot"

	// BUILDDATE contains the build date, populated during linking by goxc
	BUILDDATE = "undefined"

	// Command line flags
	help    = flag.Bool("help", false, "Displays this help")
	pidfile = flag.String("pidfile", "none", "Write our PID into the specified file")
	logfile = flag.String("logfile", "stderr", "Write out log into the specified file")

	// startTime is used to calculate uptime of Inbucket
	startTime = time.Now()

	// The file we send log output to, will be nil for stderr or stdout
	logf *os.File

	// Server instances
	smtpServer *smtpd.Server
	pop3Server *pop3d.Server
)

func main() {
	config.Version = VERSION
	config.BuildDate = BUILDDATE

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

	// Setup signal handler
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGTERM)
	go signalProcessor(sigChan)

	// Configure logging, close std* fds
	level, _ := config.Config.String("logging", "level")
	log.SetLogLevel(level)

	if *logfile != "stderr" {
		// stderr is the go logging default
		if *logfile == "stdout" {
			// set to stdout
			golog.SetOutput(os.Stdout)
		} else {
			err = openLogFile()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v", err)
				os.Exit(1)
			}
			defer closeLogFile()

			// Close std* streams to prevent accidental output, they will be redirected to
			// our logfile below
			if err := os.Stdout.Close(); err != nil {
				log.Errorf("Failed to close os.Stdout during log setup")
			}
			// Warning: this will hide panic() output
			// TODO Replace with syscall.Dup2 per https://github.com/golang/go/issues/325
			if err := os.Stderr.Close(); err != nil {
				log.Errorf("Failed to close os.Stderr during log setup")
			}
			if err := os.Stdin.Close(); err != nil {
				log.Errorf("Failed to close os.Stdin during log setup")
			}
			os.Stdout = logf
			os.Stderr = logf
		}
	}

	log.Infof("Inbucket %v (%v) starting...", config.Version, config.BuildDate)

	// Write pidfile if requested
	// TODO: Probably supposed to remove pidfile during shutdown
	if *pidfile != "none" {
		pidf, err := os.Create(*pidfile)
		if err != nil {
			log.Errorf("Failed to create %v: %v", *pidfile, err)
			os.Exit(1)
		}
		fmt.Fprintf(pidf, "%v\n", os.Getpid())
		if err := pidf.Close(); err != nil {
			log.Errorf("Failed to close PID file %v: %v", *pidfile, err)
		}
	}

	// Grab our datastore
	ds := smtpd.DefaultFileDataStore()

	// Start HTTP server
	httpd.Initialize(config.GetWebConfig(), ds)
	webui.SetupRoutes(httpd.Router)
	go httpd.Start()

	// Start POP3 server
	pop3Server = pop3d.New()
	go pop3Server.Start()

	// Startup SMTP server, block until it exits
	smtpServer = smtpd.NewServer(config.GetSMTPConfig(), ds)
	smtpServer.Start()

	// Wait for active connections to finish
	smtpServer.Drain()
	pop3Server.Drain()
}

// openLogFile creates or appends to the logfile passed on commandline
func openLogFile() error {
	// use specified log file
	var err error
	logf, err = os.OpenFile(*logfile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("Failed to create %v: %v\n", *logfile, err)
	}
	golog.SetOutput(logf)
	log.Tracef("Opened new logfile")
	return nil
}

// closeLogFile closes the current logfile
func closeLogFile() {
	log.Tracef("Closing logfile")
	// We are never in a situation where we can do anything about failing to close
	_ = logf.Close()
}

// signalProcessor is a goroutine that handles OS signals
func signalProcessor(c <-chan os.Signal) {
	for {
		sig := <-c
		switch sig {
		case syscall.SIGHUP:
			// Rotate logs if configured
			if logf != nil {
				log.Infof("Recieved SIGHUP, cycling logfile")
				closeLogFile()
				// There is nothing we can do if the log open fails
				// TODO We could panic, but that would be lame?
				_ = openLogFile()
			} else {
				log.Infof("Ignoring SIGHUP, logfile not configured")
			}
		case syscall.SIGTERM:
			// Initiate shutdown
			log.Infof("Received SIGTERM, shutting down")
			go timedExit()
			httpd.Stop()
			if smtpServer != nil {
				smtpServer.Stop()
			} else {
				log.Errorf("smtpServer was nil during shutdown")
			}
		}
	}
}

// timedExit is called as a goroutine during shutdown, it will force an exit after 15 seconds
func timedExit() {
	time.Sleep(15 * time.Second)
	log.Errorf("Inbucket clean shutdown timed out, forcing exit")
	os.Exit(0)
}

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage of inbucket [options] <conf file>:")
		flag.PrintDefaults()
	}

	expvar.Publish("uptime", expvar.Func(uptime))
}

// uptime() is published as an expvar
func uptime() interface{} {
	return time.Since(startTime) / time.Second
}
