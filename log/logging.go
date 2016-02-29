package log

import (
	"fmt"
	golog "log"
	"os"
	"strings"
)

// Level is used to indicate the severity of a log entry
type Level int

const (
	// ERROR indicates a significant problem was encountered
	ERROR Level = iota
	// WARN indicates something that may be a problem
	WARN
	// INFO indicates a purely informational log entry
	INFO
	// TRACE entries are meant for development purposes only
	TRACE
)

var (
	// MaxLevel is the highest Level we will log (max TRACE, min ERROR)
	MaxLevel = TRACE

	// logfname is the name of the logfile
	logfname string

	// logf is the file we send log output to, will be nil for stderr or stdout
	logf *os.File
)

// Initialize logging.  If logfile is equal to "stderr" or "stdout", then
// we will log to that output stream.  Otherwise the specificed file will
// opened for writing, and all log data will be placed in it.
func Initialize(logfile string) error {
	if logfile != "stderr" {
		// stderr is the go logging default
		if logfile == "stdout" {
			// set to stdout
			golog.SetOutput(os.Stdout)
		} else {
			logfname = logfile
			if err := openLogFile(); err != nil {
				return err
			}
			// Platform specific
			closeStdin()
		}
	}
	return nil
}

// SetLogLevel sets MaxLevel based on the provided string
func SetLogLevel(level string) (ok bool) {
	switch strings.ToUpper(level) {
	case "ERROR":
		MaxLevel = ERROR
	case "WARN":
		MaxLevel = WARN
	case "INFO":
		MaxLevel = INFO
	case "TRACE":
		MaxLevel = TRACE
	default:
		Errorf("Unknown log level requested: " + level)
		return false
	}
	return true
}

// Errorf logs a message to the 'standard' Logger (always), accepts format strings
func Errorf(msg string, args ...interface{}) {
	msg = "[ERROR] " + msg
	golog.Printf(msg, args...)
}

// Warnf logs a message to the 'standard' Logger if MaxLevel is >= WARN, accepts format strings
func Warnf(msg string, args ...interface{}) {
	if MaxLevel >= WARN {
		msg = "[WARN ] " + msg
		golog.Printf(msg, args...)
	}
}

// Infof logs a message to the 'standard' Logger if MaxLevel is >= INFO, accepts format strings
func Infof(msg string, args ...interface{}) {
	if MaxLevel >= INFO {
		msg = "[INFO ] " + msg
		golog.Printf(msg, args...)
	}
}

// Tracef logs a message to the 'standard' Logger if MaxLevel is >= TRACE, accepts format strings
func Tracef(msg string, args ...interface{}) {
	if MaxLevel >= TRACE {
		msg = "[TRACE] " + msg
		golog.Printf(msg, args...)
	}
}

// Rotate closes the current log file, then reopens it.  This gives an external
// log rotation system the opportunity to move the existing log file out of the
// way and have Inbucket create a new one.
func Rotate() {
	// Rotate logs if configured
	if logf != nil {
		closeLogFile()
		// There is nothing we can do if the log open fails
		_ = openLogFile()
	} else {
		Infof("Ignoring SIGHUP, logfile not configured")
	}
}

// Close the log file if we have one open
func Close() {
	if logf != nil {
		closeLogFile()
	}
}

// openLogFile creates or appends to the logfile passed on commandline
func openLogFile() error {
	// use specified log file
	var err error
	logf, err = os.OpenFile(logfname, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("Failed to create %v: %v\n", logfname, err)
	}
	golog.SetOutput(logf)
	Tracef("Opened new logfile")
	// Platform specific
	reassignStdout()
	return nil
}

// closeLogFile closes the current logfile
func closeLogFile() {
	Tracef("Closing logfile")
	// We are never in a situation where we can do anything about failing to close
	_ = logf.Close()
}
