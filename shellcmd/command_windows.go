//go:build windows

package shellcmd

import (
	"context"
	"os/exec"
)

func Command(command string) *exec.Cmd {
	return exec.Command("cmd", "/C", command)
}

func CommandContext(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, "cmd", "/C", command)
}
