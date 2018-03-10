// +build windows

package log

import (
	"os"
)

var stdOutsClosed = false

// closeStdin does nothing on Windows, it would always fail
func closeStdin() {
	// Nop
}

// reassignStdout points stdout/stderr to our logfile on systems that do not
// support the Dup2 syscall
func reassignStdout() {
	Tracef("Windows reassignStdout()")
	if !stdOutsClosed {
		// Close std* streams to prevent accidental output, they will be redirected to
		// our logfile below

		// Warning: this will hide panic() output, sorry Windows users
		if err := os.Stderr.Close(); err != nil {
			// Not considered fatal
			Errorf("Failed to close os.Stderr during log setup")
		}
		if err := os.Stdin.Close(); err != nil {
			// Not considered fatal
			Errorf("Failed to close os.Stdin during log setup")
		}
		os.Stdout = logf
		os.Stderr = logf
		stdOutsClosed = true
	}
}
