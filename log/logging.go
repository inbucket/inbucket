package log

import (
	golog "log"
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

// MaxLevel is the highest Level we will log (max TRACE, min ERROR)
var MaxLevel = TRACE

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
