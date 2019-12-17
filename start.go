// +build linux darwin

package bark

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// see w.err for any error after w.Done
func (w *Watchdog) Start() {

	signalChild := make(chan os.Signal, 1000) // try not to back up on reaping with a big channel buffer.

	signal.Notify(signalChild, syscall.SIGCHLD)

	w.needRestart = true
	var ws syscall.WaitStatus
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
			signal.Stop(signalChild) // reverse the effect of the above Notify()
		}()
		var err error

	reaploop:
		for {
			if w.needRestart {
				if w.proc != nil {
					w.proc.Release()
				}
				Q(" debug: about to start '%s'", w.PathToChildExecutable)
				w.proc, err = os.StartProcess(w.PathToChildExecutable, w.Args, &w.Attr)
				if err != nil {
					w.err = err
					return
				}
				w.curPid = w.proc.Pid
				w.needRestart = false
				w.startCount++
				Q(" Start number %d: Watchdog started pid %d / new process '%s'", w.startCount, w.proc.Pid, w.PathToChildExecutable)
			}

			select {
			case <-w.StopWatchdogAfterChildExits:
				w.exitAfterReaping = true
			case w.CurrentPid <- w.curPid:
			case <-w.TermChildAndStopWatchdog:
				Q(" TermChildAndStopWatchdog noted, exiting watchdog.Start() loop")

				err := w.proc.Signal(syscall.SIGKILL)
				if err != nil {
					err = fmt.Errorf("warning: watchdog tried to SIGKILL pid %d but got error: '%s'", w.proc.Pid, err)
					w.SetErr(err)
					log.Printf("%s", err)
					return
				}
				w.exitAfterReaping = true
				continue reaploop
			case <-w.ReqStopWatchdog:
				Q(" ReqStopWatchdog noted, exiting watchdog.Start() loop")
				return
			case <-w.RestartChild:
				Q(" debug: got <-w.RestartChild")
				err := w.proc.Signal(syscall.SIGKILL)
				if err != nil {
					err = fmt.Errorf("warning: watchdog tried to SIGKILL pid %d but got error: '%s'", w.proc.Pid, err)
					w.SetErr(err)
					log.Printf("%s", err)
					return
				}
				w.curPid = 0
				continue reaploop
			case <-signalChild:
				Q(" debug: got <-signalChild, ws is %#v", ws)

				for i := 0; i < 1000; i++ {
					pid, err := syscall.Wait4(w.proc.Pid, &ws, syscall.WNOHANG, nil)
					// pid > 0 => pid is the ID of the child that died, but
					//  there could be other children that are signalling us
					//  and not the one we in particular are waiting for.
					// pid -1 && errno == ECHILD => no new status children
					// pid -1 && errno != ECHILD => syscall interupped by signal
					// pid == 0 => no more children to wait for.
					Q(" pid=%v  ws=%#v and err == %v", pid, ws, err)
					switch {
					case err != nil:
						err = fmt.Errorf("wait4() got error back: '%s' and ws:%v", err, ws)
						log.Printf("warning in reaploop, wait4(WNOHANG) returned error: '%s'. ws=%v", err, ws)
						w.SetErr(err)
						continue reaploop
					case pid == w.proc.Pid:
						Q(" Watchdog saw OUR current w.proc.Pid %d/process '%s' finish with waitstatus: %#v.", pid, w.PathToChildExecutable, ws)
						w.ExitCode = int(ws)
						if w.exitAfterReaping {
							Q("watchdog sees exitAfterReaping. exiting now.")
							return
						}
						w.needRestart = true
						w.curPid = 0
						continue reaploop
					case pid == 0:
						// this is what we get when SIGSTOP is sent on OSX. ws == 0 in this case.
						// Note that on OSX we never get a SIGCONT signal.
						// Under WNOHANG, pid == 0 means there is nobody left to wait for,
						// so just go back to waiting for another SIGCHLD.
						Q("pid == 0 on wait4, (perhaps SIGSTOP?): nobody left to wait for, keep looping. ws = %v", ws)
						continue reaploop
					default:
						Q(" warning in reaploop: wait4() negative or not our pid, sleep and try again")
						time.Sleep(time.Millisecond)
					}
				} // end for i
				w.SetErr(fmt.Errorf("could not reap child PID %d or obtain wait4(WNOHANG)==0 even after 1000 attempts", w.proc.Pid))
				log.Printf("%s", w.err)
				return
			} // end select
		} // end for reaploop
	}()
}
