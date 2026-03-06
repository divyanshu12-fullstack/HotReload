package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hotreload/cli/builder"
	"github.com/hotreload/cli/debouncer"
	"github.com/hotreload/cli/runner"
	"github.com/hotreload/cli/watcher"
)

func main() {
	var rootDir string
	var buildCmd string
	var execCmd string

	flag.StringVar(&rootDir, "root", "", "Directory to watch recursively for file changes")
	flag.StringVar(&buildCmd, "build", "", "Shell command to build the project")
	flag.StringVar(&execCmd, "exec", "", "Shell command to run the built server")
	flag.Parse()

	if rootDir == "" || buildCmd == "" || execCmd == "" {
		slog.Error("missing required flags", "rootProvided", rootDir != "", "buildProvided", buildCmd != "", "execProvided", execCmd != "")
		fmt.Println("Usage: hotreload --root <dir> --build <build_cmd> --exec <exec_cmd>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	slog.Info("hotreload starting", "root", rootDir)

	ignoredPaths := collectIgnoredPaths(buildCmd, execCmd)
	w, err := watcher.New(rootDir, ignoredPaths...)
	if err != nil {
		slog.Error("failed to initialize watcher", "err", err)
		os.Exit(1)
	}

	if err := w.Start(); err != nil {
		slog.Error("failed to start watcher", "err", err)
		os.Exit(1)
	}
	defer w.Stop()

	d := debouncer.New(150 * time.Millisecond)
	go func() {
		for event := range w.Events {
			slog.Info("file changed, rebuilding", "file", event.Name)
			d.Input <- event
		}
	}()
	go d.Start()

	b := builder.New(buildCmd)
	r := runner.New(execCmd)

	var cancelBuild context.CancelFunc = func() {}
	var mu sync.Mutex
	rebuildQueue := make(chan struct{}, 1)
	running := false
	pending := false

	triggerRebuild := func() {
		mu.Lock()
		if running {
			pending = true
			cancelBuild()
			mu.Unlock()
			return
		}
		running = true
		mu.Unlock()

		select {
		case rebuildQueue <- struct{}{}:
		default:
		}
	}

	go func() {
		for range rebuildQueue {
			for {
				mu.Lock()
				pending = false
				ctx, cancel := context.WithCancel(context.Background())
				cancelBuild = cancel
				mu.Unlock()

				r.Stop()
				err := b.Build(ctx)

				mu.Lock()
				restartRequested := pending
				mu.Unlock()

				if err == nil && !restartRequested {
					r.Start()
				} else if err != nil && !errors.Is(err, builder.ErrBuildCanceled) {
					slog.Error("rebuild cycle failed", "err", err)
				}

				mu.Lock()
				if pending {
					mu.Unlock()
					continue
				}
				running = false
				cancelBuild = func() {}
				mu.Unlock()
				break
			}
		}
	}()

	// Initial build
	slog.Info("initial build triggered")
	triggerRebuild()

	// Shutdown handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-d.Output:
			triggerRebuild()
		case <-sigs:
			mu.Lock()
			cancelBuild()
			mu.Unlock()
			r.Stop()
			slog.Info("hotreload shutting down")
			return
		}
	}
}

func collectIgnoredPaths(buildCmd string, execCmd string) []string {
	paths := make([]string, 0, 2)

	fields := strings.Fields(buildCmd)
	for index, field := range fields {
		if field == "-o" && index+1 < len(fields) {
			paths = append(paths, resolvePath(fields[index+1]))
			break
		}
	}

	if execPath := firstCommandToken(execCmd); execPath != "" {
		paths = append(paths, resolvePath(execPath))
	}

	return uniquePaths(paths)
}

func firstCommandToken(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func resolvePath(path string) string {
	if path == "" {
		return ""
	}
	if resolved, err := filepath.Abs(path); err == nil {
		return resolved
	}
	return path
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{})
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		key := strings.ToLower(filepath.Clean(path))
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, path)
	}
	return unique
}
