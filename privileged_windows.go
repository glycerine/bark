//go:build windows
// +build windows

package bark

import (
	"os"
	"time"
)

// NewPrivilegedWatchdog creates a Watchdog structure that will run its child process
// with elevated privileges. This is primarily useful on Windows for monitoring
// privileged processes. The parent process must also have sufficient privileges
// to create elevated child processes.
func NewPrivilegedWatchdog(
	attr *os.ProcAttr,
	pathToChildExecutable string,
	args ...string) *Watchdog {

	w := NewWatchdog(attr, pathToChildExecutable, args...)
	w.isPrivileged = true
	return w
}

// NewPrivilegedOneshotReaper is like NewOneshotReaper but runs the child process
// with elevated privileges. This is primarily useful on Windows for monitoring
// privileged processes.
func NewPrivilegedOneshotReaper(
	attr *os.ProcAttr,
	pathToChildExecutable string,
	args ...string) *Watchdog {

	w := NewPrivilegedWatchdog(attr, pathToChildExecutable, args...)
	w.exitAfterReaping = true
	return w
}

// PrivilegedOneshot is like Oneshot but runs the child process with elevated privileges.
// This is primarily useful on Windows for monitoring privileged processes.
func PrivilegedOneshot(pathToProcess string, args ...string) (*Watchdog, error) {

	watcher := NewPrivilegedOneshotReaper(nil, pathToProcess, args...)
	watcher.Start()

	return watcher, nil
}

// PrivilegedOneshotAndWait is like OneshotAndWait but runs the child process
// with elevated privileges. This is primarily useful on Windows for monitoring
// privileged processes.
func PrivilegedOneshotAndWait(pathToProcess string, timeOut time.Duration, args ...string) (int, error) {
	w, _ := PrivilegedOneshot(pathToProcess, args...)
	if timeOut > 0 {
		select {
		case <-time.After(timeOut):
			return 0, TimedOut
		case <-w.Done:
		}
	} else {
		<-w.Done
	}
	return w.ExitCode, nil
}

// StartPrivilegedAndWatch is like StartAndWatch but runs the child process
// with elevated privileges. This is primarily useful on Windows for monitoring
// privileged processes.
func StartPrivilegedAndWatch(pathToProcess string, args ...string) (*Watchdog, error) {

	// start our child; restart it if it dies.
	watcher := NewPrivilegedWatchdog(nil, pathToProcess, args...)
	watcher.Start()

	return watcher, nil
}
