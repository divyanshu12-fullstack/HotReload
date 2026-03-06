package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherIgnoresExplicitOutputPath(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "out")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ignoredBinary := filepath.Join(outputDir, "server.exe")

	w, err := New(root, ignoredBinary)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(ignoredBinary, []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-w.Events:
		t.Fatalf("expected ignored output path to produce no event, got %s", event.Name)
	case <-time.After(250 * time.Millisecond):
	}
}

func TestWatcherAddsNewDirectoriesRecursively(t *testing.T) {
	root := t.TempDir()
	w, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	newDir := filepath.Join(root, "nested", "deep")
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatal(err)
	}

	time.Sleep(250 * time.Millisecond)

	filePath := filepath.Join(newDir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(time.Second)
	for {
		select {
		case event := <-w.Events:
			if normalizePath(event.Name) == normalizePath(filePath) {
				return
			}
		case <-deadline:
			t.Fatal("expected event from file inside dynamically created directory")
		}
	}
}

func TestWatcherRemovesDeletedDirectories(t *testing.T) {
	root := t.TempDir()
	childDir := filepath.Join(root, "child")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}

	w, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(childDir); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		w.mu.Lock()
		_, exists := w.watchedDirs[normalizePath(childDir)]
		w.mu.Unlock()
		if !exists {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("expected deleted directory to be removed from watcher registry")
}
