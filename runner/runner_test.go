package runner

import (
	"testing"
	"time"
)

func TestRunnerStartStop(t *testing.T) {
	// Simple command that just sleeps, giving us a chance to test killing it.
	r := New("sleep 10")
	if r.cmdString == "" {
		// on windows sleep might not be perfectly available natively without path, but let's try
	}

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
