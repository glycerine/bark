package bark

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func Test100WatchdogShouldNoticeAndRestartChild(t *testing.T) {

	cv.Convey("our Watchdog goroutine should be able to restart the child process upon request", t, func() {
		watcher := NewWatchdog(nil, "./testcmd/sleep50")
		watcher.Start()

		var err error
		testOver := time.After(6000 * time.Millisecond)

	testloop:
		for {
			select {
			case <-testOver:
				fmt.Printf("\n testOver timeout fired.\n")
				err = watcher.Stop()
				break testloop

				// 3 msec can overwhelm darwin, lets try 300 msec. get 18 starts on darwin.
				//case <-time.After(3 * time.Millisecond):
			case <-time.After(300 * time.Millisecond):
				VPrintf("watch_test: after 300 milliseconds: requesting restart of child process.\n")
				watcher.RestartChild <- true
			case <-watcher.Done:
				fmt.Printf("\n watcher.Done fired.\n")
				err = watcher.GetErr()
				fmt.Printf("\n watcher.Done, with err = '%v'\n", err)
				break testloop
			}
		}
		panicOn(err)
		// getting 14-27 starts on OSX in 500ms. But windows needs more time so require 3.
		fmt.Printf("\n done after %d starts.\n", watcher.startCount)
		cv.So(watcher.startCount, cv.ShouldBeGreaterThan, 2)
	})
}

func Test200WatchdogTerminatesUponRequest(t *testing.T) {

	cv.Convey("our Watchdog goroutine should terminate its child process and stop upon request", t, func() {

		watcher := NewWatchdog(nil, "./testcmd/sleep50", "arg1", "arg2")
		watcher.Start()

		sleepDur := 10 * time.Millisecond
		time.Sleep(sleepDur)

		//P("getting pid")
		pid := <-watcher.CurrentPid
		//P("back from getting pid")
		if pid <= 0 {
			panic("error: pid was <= 0 implying process did not start")
		}

		//P("about to send term request")
		watcher.TermChildAndStopWatchdog <- true
		//P("done sending term request")
		err := WaitForShutdownWithTimeout(pid, time.Millisecond*100)
		panicOn(err)
		//P("done with wait for shutdown")
		cv.So(err, cv.ShouldEqual, nil)

		<-watcher.Done
		//P("shutdown complete")
	})
}

func Test300OneshotReaperTerminatesUponRequest(t *testing.T) {

	cv.Convey("our OneshotReaper goroutine should terminate when its child exits", t, func() {

		// ignore errors; signal file probably not there.
		os.Remove("signal.sleep0.done")

		watcher := NewOneshotReaper(nil, "./testcmd/sleep0")
		watcher.Start()

		pauseDur := 1000 * time.Millisecond
		if runtime.GOOS == "windows" {
			pauseDur = 4 * time.Second
		}
		select {
		case <-time.After(pauseDur):
			panic(fmt.Sprintf("oneshot or our sleep0 child did not exit after %v", pauseDur))
		case <-watcher.Done:
			// okay.
		}
		//P("shutdown complete")
	})
}

func Test500OneshotAndWaitGetsExitCodesCorrectly(t *testing.T) {
	if runtime.GOOS == "windows" {
		return // probably not going to work on windows
	}

	cv.Convey("our OneshotAndWait goroutine should return our child's exit code", t, func() {

		exitCode, _ := OneshotAndWait("./testcmd/exit42", 0)
		fmt.Printf("from exit42, we got 0x%x\n", exitCode)
		cv.So(exitCode, cv.ShouldEqual, 42<<8)

		exitCode, _ = OneshotAndWait("./testcmd/exit44", 0)
		cv.So(exitCode, cv.ShouldEqual, 44<<8)

		exitCode, _ = OneshotAndWait("./testcmd/exit43", 0)
		cv.So(exitCode, cv.ShouldEqual, 43<<8)

		exitCode, _ = OneshotAndWait("./testcmd/exit0", 0)
		cv.So(exitCode, cv.ShouldEqual, 0)
	})
}
