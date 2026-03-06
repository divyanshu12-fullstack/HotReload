//go:build windows

package runner

import (
	"fmt"
	"os/exec"
	"syscall"
)

func setupSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func terminateProcessTree(cmd *exec.Cmd) error {
	return exec.Command("taskkill", "/PID", fmt.Sprint(cmd.Process.Pid), "/T").Run()
}

func forceKillProcessTree(cmd *exec.Cmd) error {
	return exec.Command("taskkill", "/PID", fmt.Sprint(cmd.Process.Pid), "/T", "/F").Run()
}
