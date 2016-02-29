// +build !windows

package log

import (
	"golang.org/x/sys/unix"
	"os"
)

// closeStdin will close stdin on Unix platforms - this is standard practice
// for daemons
func closeStdin() {
	if err := os.Stdin.Close(); err != nil {
		// Not a fatal error
		Errorf("Failed to close os.Stdin during log setup")
	}
}

// reassignStdout points stdout/stderr to our logfile on systems that support
// the Dup2 syscall per https://github.com/golang/go/issues/325
func reassignStdout() {
	Tracef("Unix reassignStdout()")
	if err := unix.Dup2(int(logf.Fd()), 1); err != nil {
		// Not considered fatal
		Errorf("Failed to re-assign stdout to logfile: %v", err)
	}
	if err := unix.Dup2(int(logf.Fd()), 2); err != nil {
		// Not considered fatal
		Errorf("Failed to re-assign stderr to logfile: %v", err)
	}
}
