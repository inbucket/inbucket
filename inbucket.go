// main is the inbucket daemon launcher
package main

import (
	"context"
	"expvar"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/filestore"
	"github.com/jhillyerd/inbucket/httpd"
	"github.com/jhillyerd/inbucket/log"
	"github.com/jhillyerd/inbucket/msghub"
	"github.com/jhillyerd/inbucket/pop3d"
	"github.com/jhillyerd/inbucket/rest"
	"github.com/jhillyerd/inbucket/smtpd"
	"github.com/jhillyerd/inbucket/webui"
)

var (
	// version contains the build version number, populated during linking
	version = "undefined"

	// date contains the build date, populated during linking
	date = "undefined"

	// Command line flags
	help    = flag.Bool("help", false, "Displays this help")
	pidfile = flag.String("pidfile", "none", "Write our PID into the specified file")
	logfile = flag.String("logfile", "stderr", "Write out log into the specified file")

	// shutdownChan - close it to tell Inbucket to shut down cleanly
	shutdownChan = make(chan bool)

	// Server instances
	smtpServer *smtpd.Server
	pop3Server *pop3d.Server
)

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage of inbucket [options] <conf file>:")
		flag.PrintDefaults()
	}

	// Server uptime for status page
	startTime := time.Now()
	expvar.Publish("uptime", expvar.Func(func() interface{} {
		return time.Since(startTime) / time.Second
	}))

	// Goroutine count for status page
	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))
}

func main() {
	config.Version = version
	config.BuildDate = date

	flag.Parse()
	if *help {
		flag.Usage()
		return
	}

	// Root context
	rootCtx, rootCancel := context.WithCancel(context.Background())

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
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	// Initialize logging
	log.SetLogLevel(config.GetLogLevel())
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

	// Create message hub
	msgHub := msghub.New(rootCtx, config.GetWebConfig().MonitorHistory)

	// Grab our datastore
	ds := filestore.DefaultFileDataStore()

	// Start HTTP server
	httpd.Initialize(config.GetWebConfig(), shutdownChan, ds, msgHub)
	webui.SetupRoutes(httpd.Router)
	rest.SetupRoutes(httpd.Router)
	go httpd.Start(rootCtx)

	// Start POP3 server
	pop3Server = pop3d.New(config.GetPOP3Config(), shutdownChan, ds)
	go pop3Server.Start(rootCtx)

	// Startup SMTP server
	smtpServer = smtpd.NewServer(config.GetSMTPConfig(), shutdownChan, ds, msgHub)
	go smtpServer.Start(rootCtx)

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
		case <-shutdownChan:
			rootCancel()
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
