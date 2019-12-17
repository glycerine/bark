package bark

import (
	"errors"
	"fmt"
	"os"
	//"os/exec"
	//"runtime"
	//"strings"
	"sync"
	"time"
)

type Watchdog struct {
	Ready                       chan bool
	RestartChild                chan bool
	ReqStopWatchdog             chan bool
	TermChildAndStopWatchdog    chan bool
	Done                        chan bool
	CurrentPid                  chan int
	StopWatchdogAfterChildExits chan bool
	ExitCode                    int

	curPid int

	startCount int64

	mut      sync.Mutex
	shutdown bool

	PathToChildExecutable string
	Args                  []string
	Attr                  os.ProcAttr
	err                   error
	needRestart           bool
	proc                  *os.Process
	exitAfterReaping      bool
}

// NewWatchdog creates a Watchdog structure but
// does not launch it or its child process until
// Start() is called on it.
// The attr argument sets function attributes such
// as environment and open files; see os.ProcAttr for details.
// Also, attr can be nil here, in which case and
// empty os.ProcAttr will be supplied to the
// os.StartProcess() call.
func NewWatchdog(
	attr *os.ProcAttr,
	pathToChildExecutable string,
	args ...string) *Watchdog {

	if FileExists(pathToChildExecutable) {
		//P("good, path '%v' exists.", pathToChildExecutable)
	} else {
		panic(fmt.Sprintf("could not find path '%v'", pathToChildExecutable))
	}

	cpOfArgs := make([]string, 0)
	for i := range args {
		cpOfArgs = append(cpOfArgs, args[i])
	}

	w := &Watchdog{
		PathToChildExecutable:       pathToChildExecutable,
		Args:                        cpOfArgs,
		Ready:                       make(chan bool),
		RestartChild:                make(chan bool),
		ReqStopWatchdog:             make(chan bool),
		TermChildAndStopWatchdog:    make(chan bool),
		StopWatchdogAfterChildExits: make(chan bool),
		Done:                        make(chan bool),
		CurrentPid:                  make(chan int),
	}

	if attr != nil {
		w.Attr = *attr
	}
	return w
}

// NewOneshotReaper() is just like NewWatchdog,
// except that it also does two things:
//
// a) sets a flag so that the
// watchdog won't restart the child process
// after it finishes.
//
// and
//
// b) the watchdog goroutine will itself exit
// once the child exists--after cleaning up.
// Cleanup means we don't leave
// zombies and we call go's os.Process.Release() to
// release associated resources after the child
// exits.
//
// You still need to call Start(), just like
// after NewWatchdog().
//
func NewOneshotReaper(
	attr *os.ProcAttr,
	pathToChildExecutable string,
	args ...string) *Watchdog {

	w := NewWatchdog(attr, pathToChildExecutable, args...)
	w.exitAfterReaping = true
	return w
}

// Oneshot() combines NewOneshotReaper() and Start().
// In other words, we start the child once. We don't
// restart once it finishes. Instead we just reap and
// cleanup. The returned pointer's Done channel will
// be closed when the child process and watchdog
// goroutine have finished.
func Oneshot(pathToProcess string, args ...string) (*Watchdog, error) {

	watcher := NewOneshotReaper(nil, pathToProcess, args...)
	watcher.Start()

	return watcher, nil
}

var TimedOut = errors.New("TimedOut")

// OneshotAndWait runs the command pathToProcess with the
// given args, and returns the exit code of the process.
// You may need to divide the exit code by 256 to get what you
// expect, at least on OSX.
//
// With timeOut of 0 we wait forever and return the
// exitcode and an error of nil. Otherwise we return
// 0 and the TimedOut error if we timed-out waiting
// for the child process to finish.
func OneshotAndWait(pathToProcess string, timeOut time.Duration, args ...string) (int, error) {
	w, _ := Oneshot(pathToProcess, args...)
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

// StartAndWatch() is the convenience/main entry API.
// pathToProcess should give the path to the executable within
// the filesystem. If it dies it will be restarted by
// the Watchdog.
func StartAndWatch(pathToProcess string, args ...string) (*Watchdog, error) {

	// start our child; restart it if it dies.
	watcher := NewWatchdog(nil, pathToProcess, args...)
	watcher.Start()

	return watcher, nil
}

func (w *Watchdog) AlreadyDone() bool {
	select {
	case <-w.Done:
		return true
	default:
		return false
	}
}
func (w *Watchdog) Stop() error {
	if w.AlreadyDone() {
		// once Done, w.err is immutable, so we don't need to lock.
		return w.err
	}
	w.mut.Lock()
	if w.shutdown {
		defer w.mut.Unlock()
		return w.err
	}
	w.mut.Unlock()

	close(w.ReqStopWatchdog)
	<-w.Done
	// don't wait for Done while holding the lock,
	// as that is deadlock prone.

	w.mut.Lock()
	defer w.mut.Unlock()
	w.shutdown = true
	return w.err
}

func (w *Watchdog) SetErr(err error) {
	w.mut.Lock()
	defer w.mut.Unlock()
	w.err = err
}

func (w *Watchdog) GetErr() error {
	w.mut.Lock()
	defer w.mut.Unlock()
	return w.err
}
