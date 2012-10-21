package inbucket

import (
	"log"
)

type LogLevel int

const (
	ERROR LogLevel = iota
	WARN
	INFO
	TRACE
)

var MaxLogLevel LogLevel = TRACE

// Error logs a message to the 'standard' Logger (always)
func Error(msg string, args ...interface{}) {
	msg = "[ERROR] " + msg
	log.Printf(msg, args...)
}

// Warn logs a message to the 'standard' Logger if MaxLogLevel is >= WARN
func Warn(msg string, args ...interface{}) {
	if MaxLogLevel >= WARN {
		msg = "[WARN ] " + msg
		log.Printf(msg, args...)
	}
}

// Info logs a message to the 'standard' Logger if MaxLogLevel is >= INFO
func Info(msg string, args ...interface{}) {
	if MaxLogLevel >= INFO {
		msg = "[INFO ] " + msg
		log.Printf(msg, args...)
	}
}

// Trace logs a message to the 'standard' Logger if MaxLogLevel is >= TRACE
func Trace(msg string, args ...interface{}) {
	if MaxLogLevel >= TRACE {
		msg = "[TRACE] " + msg
		log.Printf(msg, args...)
	}
}
