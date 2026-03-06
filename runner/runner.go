package runner

import (
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/hotreload/cli/shellcmd"
)

type Runner struct {
	cmdString string
	cmd       *exec.Cmd
	done      chan struct{}
	stopping  *exec.Cmd
	mu        sync.Mutex
	startTime time.Time
}

func New(cmdString string) *Runner {
	return &Runner{
		cmdString: cmdString,
	}
}

func (r *Runner) Start() {
	if r.cmdString == "" {
		return
	}

	cmd := shellcmd.Command(r.cmdString)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Platform specific setup for process groups
	setupSysProcAttr(cmd)

	slog.Info("server starting", "cmd", r.cmdString)
	startTime := time.Now()

	err := cmd.Start()
	if err != nil {
		slog.Error("server failed to start", "err", err)
		return
	}

	r.mu.Lock()
	r.cmd = cmd
	r.done = make(chan struct{})
	r.stopping = nil
	r.startTime = startTime
	done := r.done
	r.mu.Unlock()

	go r.wait(cmd, done, startTime)
}

func (r *Runner) wait(cmd *exec.Cmd, done chan struct{}, startTime time.Time) {
	err := cmd.Wait()
	close(done)

	r.mu.Lock()
	intentionalStop := r.stopping == cmd
	if r.cmd == cmd {
		r.cmd = nil
		r.done = nil
	}
	if intentionalStop {
		r.stopping = nil
	}
	r.mu.Unlock()

	if err != nil {
		slog.Debug("server exited", "err", err)
	}

	if !intentionalStop && time.Since(startTime) < time.Second {
		slog.Warn("server crashed immediately, backing off", "wait", "3s")
		time.Sleep(3 * time.Second)
	}
}

func (r *Runner) Stop() {
	r.mu.Lock()
	cmd := r.cmd
	done := r.done
	if cmd == nil || cmd.Process == nil {
		r.mu.Unlock()
		return
	}
	r.stopping = cmd
	r.mu.Unlock()

	slog.Info("stopping server")

	if err := terminateProcessTree(cmd); err != nil {
		slog.Debug("graceful stop failed", "err", err)
	}

	select {
	case <-done:
		slog.Info("server stopped")
		return
	case <-time.After(3 * time.Second):
	}

	if err := forceKillProcessTree(cmd); err != nil {
		slog.Debug("force kill failed", "err", err)
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
	}

	slog.Info("server stopped")
}
