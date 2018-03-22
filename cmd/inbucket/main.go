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

	"github.com/jhillyerd/inbucket/pkg/config"
	"github.com/jhillyerd/inbucket/pkg/log"
	"github.com/jhillyerd/inbucket/pkg/message"
	"github.com/jhillyerd/inbucket/pkg/msghub"
	"github.com/jhillyerd/inbucket/pkg/policy"
	"github.com/jhillyerd/inbucket/pkg/rest"
	"github.com/jhillyerd/inbucket/pkg/server/pop3"
	"github.com/jhillyerd/inbucket/pkg/server/smtp"
	"github.com/jhillyerd/inbucket/pkg/server/web"
	"github.com/jhillyerd/inbucket/pkg/storage"
	"github.com/jhillyerd/inbucket/pkg/storage/file"
	"github.com/jhillyerd/inbucket/pkg/webui"
)

var (
	// version contains the build version number, populated during linking.
	version = "undefined"

	// date contains the build date, populated during linking.
	date = "undefined"
)

func init() {
	// Server uptime for status page.
	startTime := time.Now()
	expvar.Publish("uptime", expvar.Func(func() interface{} {
		return time.Since(startTime) / time.Second
	}))

	// Goroutine count for status page.
	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))
}

func main() {
	// Command line flags.
	help := flag.Bool("help", false, "Displays this help")
	pidfile := flag.String("pidfile", "", "Write our PID into the specified file")
	logfile := flag.String("logfile", "stderr", "Write out log into the specified file")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: inbucket [options]")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "")
		config.Usage()
	}
	flag.Parse()
	if *help {
		flag.Usage()
		return
	}
	// Process configuration.
	config.Version = version
	config.BuildDate = date
	conf, err := config.Process()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}
	// Setup signal handler.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	// Initialize logging.
	log.SetLogLevel(conf.LogLevel)
	if err := log.Initialize(*logfile); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	defer log.Close()
	log.Infof("Inbucket %v (%v) starting...", config.Version, config.BuildDate)
	// Write pidfile if requested.
	if *pidfile != "" {
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
	// Configure internal services.
	rootCtx, rootCancel := context.WithCancel(context.Background())
	shutdownChan := make(chan bool)
	msgHub := msghub.New(rootCtx, conf.Web.MonitorHistory)
	store := file.New(conf.Storage)
	addrPolicy := &policy.Addressing{Config: conf.SMTP}
	mmanager := &message.StoreManager{Store: store, Hub: msgHub}
	// Start Retention scanner.
	retentionScanner := storage.NewRetentionScanner(conf.Storage, store, shutdownChan)
	retentionScanner.Start()
	// Start HTTP server.
	web.Initialize(conf, shutdownChan, mmanager, msgHub)
	webui.SetupRoutes(web.Router)
	rest.SetupRoutes(web.Router)
	go web.Start(rootCtx)
	// Start POP3 server.
	pop3Server := pop3.New(conf.POP3, shutdownChan, store)
	go pop3Server.Start(rootCtx)
	// Start SMTP server.
	smtpServer := smtp.NewServer(conf.SMTP, shutdownChan, mmanager, addrPolicy)
	go smtpServer.Start(rootCtx)
	// Loop forever waiting for signals or shutdown channel.
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
	// Wait for active connections to finish.
	go timedExit(*pidfile)
	smtpServer.Drain()
	pop3Server.Drain()
	retentionScanner.Join()
	removePIDFile(*pidfile)
}

// removePIDFile removes the PID file if created.
func removePIDFile(pidfile string) {
	if pidfile != "" {
		if err := os.Remove(pidfile); err != nil {
			log.Errorf("Failed to remove %q: %v", pidfile, err)
		}
	}
}

// timedExit is called as a goroutine during shutdown, it will force an exit after 15 seconds.
func timedExit(pidfile string) {
	time.Sleep(15 * time.Second)
	log.Errorf("Clean shutdown took too long, forcing exit")
	removePIDFile(pidfile)
	os.Exit(0)
}
