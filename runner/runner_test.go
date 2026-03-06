package runner

import (
	"runtime"
	"testing"
	"time"
)

func TestRunnerStartStop(t *testing.T) {
	r := New(longRunningCommand())

	r.Start()

	// give it a tiny bit of time to start
	time.Sleep(50 * time.Millisecond)

	r.mu.Lock()
	cmd := r.cmd
	done := r.done
	r.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		t.Fatal("process was not started")
	}

	// Test Stop
	r.Stop()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for process to exit")
	}

	if cmd.ProcessState == nil {
		t.Fatal("process state is nil, suggesting process is still running or wasn't reaped")
	}
	if !cmd.ProcessState.Exited() {
		t.Fatal("process did not exit")
	}
}

func longRunningCommand() string {
	if runtime.GOOS == "windows" {
		return "ping -n 30 127.0.0.1 > NUL"
	}
	return "sleep 30"
}
