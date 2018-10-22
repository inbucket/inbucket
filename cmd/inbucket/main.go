// main is the inbucket daemon launcher
package main

import (
	"bufio"
	"context"
	"expvar"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/jhillyerd/inbucket/pkg/config"
	"github.com/jhillyerd/inbucket/pkg/message"
	"github.com/jhillyerd/inbucket/pkg/msghub"
	"github.com/jhillyerd/inbucket/pkg/policy"
	"github.com/jhillyerd/inbucket/pkg/rest"
	"github.com/jhillyerd/inbucket/pkg/server/pop3"
	"github.com/jhillyerd/inbucket/pkg/server/smtp"
	"github.com/jhillyerd/inbucket/pkg/server/web"
	"github.com/jhillyerd/inbucket/pkg/storage"
	"github.com/jhillyerd/inbucket/pkg/storage/file"
	"github.com/jhillyerd/inbucket/pkg/storage/mem"
	"github.com/jhillyerd/inbucket/pkg/webui"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	// Register storage implementations.
	storage.Constructors["file"] = file.New
	storage.Constructors["memory"] = mem.New
}

func main() {
	// Command line flags.
	help := flag.Bool("help", false, "Displays help on flags and env variables.")
	pidfile := flag.String("pidfile", "", "Write our PID into the specified file.")
	logfile := flag.String("logfile", "stderr", "Write out log into the specified file.")
	logjson := flag.Bool("logjson", false, "Logs are written in JSON format.")
	netdebug := flag.Bool("netdebug", false, "Dump SMTP & POP3 network traffic to stdout.")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: inbucket [options]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *help {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "")
		config.Usage()
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
	if *netdebug {
		conf.POP3.Debug = true
		conf.SMTP.Debug = true
	}
	// Logger setup.
	closeLog, err := openLog(conf.LogLevel, *logfile, *logjson)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Log error: %v\n", err)
		os.Exit(1)
	}
	startupLog := log.With().Str("phase", "startup").Logger()
	// Setup signal handler.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	// Initialize logging.
	startupLog.Info().Str("version", config.Version).Str("buildDate", config.BuildDate).
		Msg("Inbucket starting")
	// Write pidfile if requested.
	if *pidfile != "" {
		pidf, err := os.Create(*pidfile)
		if err != nil {
			startupLog.Fatal().Err(err).Str("path", *pidfile).Msg("Failed to create pidfile")
		}
		fmt.Fprintf(pidf, "%v\n", os.Getpid())
		if err := pidf.Close(); err != nil {
			startupLog.Fatal().Err(err).Str("path", *pidfile).Msg("Failed to close pidfile")
		}
	}
	// Configure internal services.
	rootCtx, rootCancel := context.WithCancel(context.Background())
	shutdownChan := make(chan bool)
	store, err := storage.FromConfig(conf.Storage)
	if err != nil {
		removePIDFile(*pidfile)
		startupLog.Fatal().Err(err).Str("module", "storage").Msg("Fatal storage error")
	}
	msgHub := msghub.New(rootCtx, conf.Web.MonitorHistory)
	addrPolicy := &policy.Addressing{Config: conf}
	mmanager := &message.StoreManager{AddrPolicy: addrPolicy, Store: store, Hub: msgHub}
	// Start Retention scanner.
	retentionScanner := storage.NewRetentionScanner(conf.Storage, store, shutdownChan)
	retentionScanner.Start()
	// Start HTTP server.
	web.Initialize(conf, shutdownChan, mmanager, msgHub)
	rest.SetupRoutes(web.Router.PathPrefix("/api/").Subrouter())
	webui.SetupRoutes(web.Router)
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
			case syscall.SIGINT:
				// Shutdown requested
				log.Info().Str("phase", "shutdown").Str("signal", "SIGINT").
					Msg("Received SIGINT, shutting down")
				close(shutdownChan)
			case syscall.SIGTERM:
				// Shutdown requested
				log.Info().Str("phase", "shutdown").Str("signal", "SIGTERM").
					Msg("Received SIGTERM, shutting down")
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
	closeLog()
}

// openLog configures zerolog output, returns func to close logfile.
func openLog(level string, logfile string, json bool) (close func(), err error) {
	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		return nil, fmt.Errorf("Log level %q not one of: debug, info, warn, error", level)
	}
	close = func() {}
	var w io.Writer
	color := runtime.GOOS != "windows"
	switch logfile {
	case "stderr":
		w = os.Stderr
	case "stdout":
		w = os.Stdout
	default:
		logf, err := os.OpenFile(logfile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return nil, err
		}
		bw := bufio.NewWriter(logf)
		w = bw
		color = false
		close = func() {
			_ = bw.Flush()
			_ = logf.Close()
		}
	}
	w = zerolog.SyncWriter(w)
	if json {
		log.Logger = log.Output(w)
		return close, nil
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:     w,
		NoColor: !color,
	})
	return close, nil
}

// removePIDFile removes the PID file if created.
func removePIDFile(pidfile string) {
	if pidfile != "" {
		if err := os.Remove(pidfile); err != nil {
			log.Error().Str("phase", "shutdown").Err(err).Str("path", pidfile).
				Msg("Failed to remove pidfile")
		}
	}
}

// timedExit is called as a goroutine during shutdown, it will force an exit after 15 seconds.
func timedExit(pidfile string) {
	time.Sleep(15 * time.Second)
	removePIDFile(pidfile)
	log.Error().Str("phase", "shutdown").Msg("Clean shutdown took too long, forcing exit")
	os.Exit(0)
}
