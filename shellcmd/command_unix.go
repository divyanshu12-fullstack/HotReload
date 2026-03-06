//go:build !windows

package shellcmd

import (
	"context"
	"os/exec"
)

func Command(command string) *exec.Cmd {
	return exec.Command("sh", "-c", command)
}

func CommandContext(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, "sh", "-c", command)
}
