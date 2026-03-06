package builder

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBuildReturnsCanceledError(t *testing.T) {
	builder := New("definitely-not-a-real-command")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := builder.Build(ctx)
	if !errors.Is(err, ErrBuildCanceled) {
		t.Fatalf("expected ErrBuildCanceled, got %v", err)
	}
}

func TestBuildReturnsFailureError(t *testing.T) {
	builder := New("definitely-not-a-real-command")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := builder.Build(ctx)
	var buildErr *BuildFailureError
	if !errors.As(err, &buildErr) {
		t.Fatalf("expected BuildFailureError, got %T", err)
	}
}
