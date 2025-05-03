//go:build windows
// +build windows

package bark

import (
	"fmt"
	"testing"
	"time"

	cv "github.com/glycerine/goconvey/convey"
)

func TestPrivilegedWatchdog(t *testing.T) {
	cv.Convey("our PrivilegedWatchdog should be able to run and monitor privileged processes", t, func() {
		watcher := NewPrivilegedWatchdog(nil, "./testcmd/sleep50")
		watcher.Start()

		sleepDur := 10 * time.Millisecond
		time.Sleep(sleepDur)

		pid := <-watcher.CurrentPid
		if pid <= 0 {
			panic("error: pid was <= 0 implying process did not start")
		}

		watcher.TermChildAndStopWatchdog <- true
		err := WaitForShutdownWithTimeout(pid, time.Millisecond*100)
		panicOn(err)
		cv.So(err, cv.ShouldEqual, nil)

		<-watcher.Done
	})
}

func TestPrivilegedOneshotReaper(t *testing.T) {
	cv.Convey("our PrivilegedOneshotReaper should be able to run and monitor privileged processes", t, func() {
		watcher := NewPrivilegedOneshotReaper(nil, "./testcmd/sleep50")
		watcher.Start()

		sleepDur := 10 * time.Millisecond
		time.Sleep(sleepDur)

		pid := <-watcher.CurrentPid
		if pid <= 0 {
			panic("error: pid was <= 0 implying process did not start")
		}

		watcher.TermChildAndStopWatchdog <- true
		err := WaitForShutdownWithTimeout(pid, time.Millisecond*100)
		panicOn(err)
		cv.So(err, cv.ShouldEqual, nil)

		<-watcher.Done
	})
}

func TestPrivilegedOneshotAndWait(t *testing.T) {
	cv.Convey("our PrivilegedOneshotAndWait should be able to run and monitor privileged processes", t, func() {
		exitCode, err := PrivilegedOneshotAndWait("./testcmd/exit42", 0)
		panicOn(err)
		fmt.Printf("from exit42, we got 0x%x\n", exitCode)
		cv.So(exitCode, cv.ShouldEqual, 42<<8)

		exitCode, err = PrivilegedOneshotAndWait("./testcmd/exit44", 0)
		panicOn(err)
		cv.So(exitCode, cv.ShouldEqual, 44<<8)

		exitCode, err = PrivilegedOneshotAndWait("./testcmd/exit43", 0)
		panicOn(err)
		cv.So(exitCode, cv.ShouldEqual, 43<<8)

		exitCode, err = PrivilegedOneshotAndWait("./testcmd/exit0", 0)
		panicOn(err)
		cv.So(exitCode, cv.ShouldEqual, 0)
	})
}

func TestStartPrivilegedAndWatch(t *testing.T) {
	cv.Convey("our StartPrivilegedAndWatch should be able to run and monitor privileged processes", t, func() {
		watcher, err := StartPrivilegedAndWatch("./testcmd/sleep50")
		panicOn(err)

		sleepDur := 10 * time.Millisecond
		time.Sleep(sleepDur)

		pid := <-watcher.CurrentPid
		if pid <= 0 {
			panic("error: pid was <= 0 implying process did not start")
		}

		watcher.TermChildAndStopWatchdog <- true
		err = WaitForShutdownWithTimeout(pid, time.Millisecond*100)
		panicOn(err)
		cv.So(err, cv.ShouldEqual, nil)

		<-watcher.Done
	})
}
