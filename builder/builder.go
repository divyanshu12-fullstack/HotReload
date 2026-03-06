package builder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/hotreload/cli/shellcmd"
)

var ErrBuildCanceled = errors.New("build canceled")

type BuildFailureError struct {
	Command string
	Err     error
}

func (e *BuildFailureError) Error() string {
	return fmt.Sprintf("build command %q failed: %v", e.Command, e.Err)
}

func (e *BuildFailureError) Unwrap() error {
	return e.Err
}

type Builder struct {
	cmdString string
}

func New(cmdString string) *Builder {
	return &Builder{
		cmdString: cmdString,
	}
}

func (b *Builder) Build(ctx context.Context) error {
	if b.cmdString == "" {
		return nil
	}

	cmd := shellcmd.CommandContext(ctx, b.cmdString)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	err := cmd.Run()

	if err != nil {
		if ctx.Err() == context.Canceled {
			slog.Info("build cancelled")
			return ErrBuildCanceled
		}
		slog.Error("build failed", "err", err)
		return &BuildFailureError{Command: b.cmdString, Err: err}
	}

	duration := time.Since(start).Round(time.Millisecond)
	slog.Info("build succeeded", "duration", duration)
	return nil
}
