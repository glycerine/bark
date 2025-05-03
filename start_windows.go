//go:build windows
// +build windows

package bark

import (
	"context"
	//"fmt"
	//"log"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	CREATE_NO_WINDOW = 0x08000000
)

// see w.err for any error after w.Done
func (w *Watchdog) Start() {

	childFini := make(chan bool, 0)

	w.needRestart = true
	go func() {
		defer func() {
			if w.proc != nil {
				w.proc.Release()
			}
			close(w.Done)
			// can deadlock if we don't close(w.Done) before grabbing the mutex:
			w.mut.Lock()
			w.shutdown = true
			w.mut.Unlock()
		}()
		var err error
		var ctx context.Context
		var cancel context.CancelFunc

	reaploop:
		for {
			if w.needRestart {
				if w.proc != nil {
					w.proc.Release()
				}
				//Q(" debug: about to start '%s'", w.PathToChildExecutable)

				//vv("see if we can actually execute cmd under cygwin:")
				var c *exec.Cmd
				bash := "C:\\cygwin64\\bin\\bash.exe"
				ctx, cancel = context.WithCancel(context.Background())
				_ = ctx
				_ = cancel
				//vv("upon start, cancel is now '%p'", cancel)
				if strings.Contains(w.PathToChildExecutable, "/testcmd/") && FileExists(bash) {
					//vv("running under cmd and then bash chain: '%v'", bash)
					c = exec.CommandContext(ctx, "cmd", "/C", bash, "-c", w.PathToChildExecutable)
				} else {
					//vv("running raw")
					c = exec.CommandContext(ctx, w.PathToChildExecutable, w.Args...)
				}

				// Set up process creation flags for elevated privileges if needed
				if w.isPrivileged {
					c.SysProcAttr = &syscall.SysProcAttr{
						CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | CREATE_NO_WINDOW,
						// Use the current process's token for elevation
						Token: syscall.Token(0),
					}
				}

				err = c.Start()
				if err != nil {
					w.err = err
					return
				}
				time.Sleep(2 * time.Second)
				//vv("we appear to have started child successfully. pid='%v'", c.Process.Pid)
				w.proc = c.Process

				go func(c *exec.Cmd) {
					//vv("goro is about to c.Wait() on child to finish")
					err := c.Wait()
					if err != nil {
						if exitErr, ok := err.(*exec.ExitError); ok {
							// Get the exit code from the ExitError
							if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
								w.ExitCode = status.ExitStatus()
							}
						}
					} else {
						// Process exited normally
						w.ExitCode = 0
					}
					//vv("child Wait() done")
					childFini <- true
					//vv("child wait finished for c.Process.Pid='%v'", c.Process.Pid)
				}(c)

				w.curPid = w.proc.Pid
				w.needRestart = false
				w.startCount++
				//vv(" Start number %d: Watchdog started pid %d / new process '%s'", w.startCount, w.proc.Pid, w.PathToChildExecutable)
			}

			select {
			case <-w.StopWatchdogAfterChildExits:
				w.exitAfterReaping = true
			case w.CurrentPid <- w.curPid:
			case <-w.TermChildAndStopWatchdog:
				//vv("TermChildAndStopWatchdog noted, exiting watchdog.Start() loop")

				if cancel != nil {
					//vv("doing cancel()")
					cancel()
					//vv("back from cancel")
				} else {
					//vv("could not cancel() b/c it was nil")
				}
				//vv("setting cancel to nil")
				cancel = nil
				/*err := w.proc.Kill() // Signal(syscall.SIGKILL)
				if err != nil {
					// getting "TerminateProcess: Access is denied."
					err = fmt.Errorf("warning: watchdog tried to SIGKILL pid %d but got error: '%s'", w.proc.Pid, err)
					w.SetErr(err)
					log.Printf("%s", err)
					return
				}*/
				w.exitAfterReaping = true
				continue reaploop
			case <-w.ReqStopWatchdog:
				//vv(" ReqStopWatchdog noted, exiting watchdog.Start() loop")
				return
			case <-w.RestartChild:
				//vv("debug: got <-w.RestartChild")
				if cancel != nil {
					//vv("cancel was not nil in RestartChild")
					cancel()
				}
				//vv("RestartChild is now setting cancel to nil")
				cancel = nil
				/*err := w.proc.Signal(syscall.SIGKILL)
				if err != nil {
					err = fmt.Errorf("warning: watchdog tried to SIGKILL pid %d but got error: '%s'", w.proc.Pid, err)
					w.SetErr(err)
					log.Printf("%s", err)
					return
				}*/

				w.curPid = 0
				continue reaploop
			case <-childFini:
				//vv(" debug: got <-childFini")
				if w.exitAfterReaping {
					//vv("watchdog sees exitAfterReaping. exiting now.")
					return
				}
				w.needRestart = true
				w.curPid = 0
				continue reaploop
			} // end select
		} // end for reaploop
	}()
}
