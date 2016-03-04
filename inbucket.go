// main is the inbucket daemon launcher
package main

import (
	"expvar"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/httpd"
	"github.com/jhillyerd/inbucket/log"
	"github.com/jhillyerd/inbucket/pop3d"
	"github.com/jhillyerd/inbucket/rest"
	"github.com/jhillyerd/inbucket/smtpd"
	"github.com/jhillyerd/inbucket/webui"
)

var (
	// VERSION contains the build version number, populated during linking by goxc
	VERSION = "1.1.0-rc1"

	// BUILDDATE contains the build date, populated during linking by goxc
	BUILDDATE = "undefined"

	// Command line flags
	help    = flag.Bool("help", false, "Displays this help")
	pidfile = flag.String("pidfile", "none", "Write our PID into the specified file")
	logfile = flag.String("logfile", "stderr", "Write out log into the specified file")

	// startTime is used to calculate uptime of Inbucket
	startTime = time.Now()

	// shutdownChan - close it to tell Inbucket to shut down cleanly
	shutdownChan = make(chan bool)

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
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	// Initialize logging
	level, _ := config.Config.String("logging", "level")
	log.SetLogLevel(level)
	if err := log.Initialize(*logfile); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	defer log.Close()

	log.Infof("Inbucket %v (%v) starting...", config.Version, config.BuildDate)

	// Write pidfile if requested
	if *pidfile != "none" {
		pidf, err := os.Create(*pidfile)
		if err != nil {
			log.Errorf("Failed to create %q: %v", *pidfile, err)
			os.Exit(1)
		}
		fmt.Fprintf(pidf, "%v\n", os.Getpid())
		if err := pidf.Close(); err != nil {
			log.Errorf("Failed to close PID file %q: %v", *pidfile, err)
		}
	}

	// Grab our datastore
	ds := smtpd.DefaultFileDataStore()

	// Start HTTP server
	httpd.Initialize(config.GetWebConfig(), ds, shutdownChan)
	webui.SetupRoutes(httpd.Router)
	rest.SetupRoutes(httpd.Router)
	go httpd.Start()

	// Start POP3 server
	// TODO pass datastore
	pop3Server = pop3d.New(shutdownChan)
	go pop3Server.Start()

	// Startup SMTP server
	smtpServer = smtpd.NewServer(config.GetSMTPConfig(), ds, shutdownChan)
	go smtpServer.Start()

	// Loop forever waiting for signals or shutdown channel
signalLoop:
	for {
		select {
		case sig := <-sigChan:
			switch sig {
			case syscall.SIGHUP:
				log.Infof("Recieved SIGHUP, cycling logfile")
				log.Rotate()
			case syscall.SIGINT:
				// Shutdown requested
				log.Infof("Received SIGINT, shutting down")
				close(shutdownChan)
			case syscall.SIGTERM:
				// Shutdown requested
				log.Infof("Received SIGTERM, shutting down")
				close(shutdownChan)
			}
		case _ = <-shutdownChan:
			break signalLoop
		}
	}

	// Wait for active connections to finish
	go timedExit()
	smtpServer.Drain()
	pop3Server.Drain()

	removePIDFile()
}

// removePIDFile removes the PID file if created
func removePIDFile() {
	if *pidfile != "none" {
		if err := os.Remove(*pidfile); err != nil {
			log.Errorf("Failed to remove %q: %v", *pidfile, err)
		}
	}
}

// timedExit is called as a goroutine during shutdown, it will force an exit
// after 15 seconds
func timedExit() {
	time.Sleep(15 * time.Second)
	log.Errorf("Clean shutdown took too long, forcing exit")
	removePIDFile()
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
