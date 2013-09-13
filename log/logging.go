package log

import (
	"log"
	"strings"
)

type LogLevel int

const (
	ERROR LogLevel = iota
	WARN
	INFO
	TRACE
)

var MaxLogLevel LogLevel = TRACE

// SetLogLevel sets MaxLogLevel based on the provided string
func SetLogLevel(level string) (ok bool) {
	switch strings.ToUpper(level) {
	case "ERROR":
		MaxLogLevel = ERROR
	case "WARN":
		MaxLogLevel = WARN
	case "INFO":
		MaxLogLevel = INFO
	case "TRACE":
		MaxLogLevel = TRACE
	default:
		LogError("Unknown log level requested: %v", level)
		return false
	}
	return true
}

// Error logs a message to the 'standard' Logger (always)
func LogError(msg string, args ...interface{}) {
	msg = "[ERROR] " + msg
	log.Printf(msg, args...)
}

// Warn logs a message to the 'standard' Logger if MaxLogLevel is >= WARN
func LogWarn(msg string, args ...interface{}) {
	if MaxLogLevel >= WARN {
		msg = "[WARN ] " + msg
		log.Printf(msg, args...)
	}
}

// Info logs a message to the 'standard' Logger if MaxLogLevel is >= INFO
func LogInfo(msg string, args ...interface{}) {
	if MaxLogLevel >= INFO {
		msg = "[INFO ] " + msg
		log.Printf(msg, args...)
	}
}

// Trace logs a message to the 'standard' Logger if MaxLogLevel is >= TRACE
func LogTrace(msg string, args ...interface{}) {
	if MaxLogLevel >= TRACE {
		msg = "[TRACE] " + msg
		log.Printf(msg, args...)
	}
}
