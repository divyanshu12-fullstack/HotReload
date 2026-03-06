package watcher

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	root         string
	watcher      *fsnotify.Watcher
	Events       chan fsnotify.Event
	ignoredPaths []string
	watchedDirs  map[string]struct{}
	mu           sync.Mutex
}

func New(root string, ignoredPaths ...string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		w.Close()
		return nil, err
	}

	normalizedIgnored := make([]string, 0, len(ignoredPaths))
	for _, path := range ignoredPaths {
		if path == "" {
			continue
		}
		normalizedIgnored = append(normalizedIgnored, normalizePath(path))
	}

	wt := &Watcher{
		root:         absRoot,
		watcher:      w,
		Events:       make(chan fsnotify.Event, 100),
		ignoredPaths: normalizedIgnored,
		watchedDirs:  make(map[string]struct{}),
	}

	return wt, nil
}

func (w *Watcher) Start() error {
	if err := w.addDirRecursive(w.root); err != nil {
		return err
	}

	slog.Info("watching directories", "count", w.directoryCount())
	if w.directoryCount() > 1000 {
		slog.Warn("watched directory count exceeds 1000. Consider increasing OS inotify limit")
	}

	go w.readEvents()
	return nil
}

func (w *Watcher) addDirRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if w.shouldIgnoreDir(d.Name()) {
				return filepath.SkipDir
			}

			normalizedPath := normalizePath(path)
			if w.shouldIgnorePath(normalizedPath) {
				return filepath.SkipDir
			}

			w.mu.Lock()
			_, alreadyWatching := w.watchedDirs[normalizedPath]
			w.mu.Unlock()
			if alreadyWatching {
				return nil
			}

			if err := w.watcher.Add(normalizedPath); err == nil {
				w.mu.Lock()
				w.watchedDirs[normalizedPath] = struct{}{}
				count := len(w.watchedDirs)
				w.mu.Unlock()
				if count > 1000 {
					slog.Warn("watched directory count exceeds 1000. Consider increasing OS inotify limit", "count", count)
				}
			}
		}
		return nil
	})
}

func (w *Watcher) directoryCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.watchedDirs)
}

func (w *Watcher) shouldIgnorePath(path string) bool {
	normalizedPath := normalizePath(path)
	for _, ignoredPath := range w.ignoredPaths {
		if pathEquals(normalizedPath, ignoredPath) {
			return true
		}
	}
	return false
}

func (w *Watcher) shouldIgnoreDir(name string) bool {
	ignored := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		"bin":          true,
		"dist":         true,
		"build":        true,
	}
	return ignored[name]
}

func (w *Watcher) shouldIgnoreFile(name string) bool {
	if strings.HasPrefix(name, "#") || strings.HasPrefix(name, ".#") {
		return true
	}
	if name == "4913" || name == ".DS_Store" {
		return true
	}
	ext := filepath.Ext(name)
	ignoredExts := map[string]bool{
		".swp": true,
		".swx": true,
		".swo": true,
		"~":    true,
		".tmp": true,
		".log": true,
	}
	if ignoredExts[ext] {
		return true
	}
	if strings.HasSuffix(name, "~") {
		return true
	}
	return false
}

func (w *Watcher) removeWatchedDir(path string) {
	normalizedPath := normalizePath(path)

	w.mu.Lock()
	pathsToRemove := make([]string, 0)
	for watchedPath := range w.watchedDirs {
		if pathEquals(watchedPath, normalizedPath) || strings.HasPrefix(watchedPath, normalizedPath+string(os.PathSeparator)) {
			pathsToRemove = append(pathsToRemove, watchedPath)
		}
	}
	for _, watchedPath := range pathsToRemove {
		delete(w.watchedDirs, watchedPath)
	}
	w.mu.Unlock()

	for _, watchedPath := range pathsToRemove {
		if err := w.watcher.Remove(watchedPath); err != nil {
			slog.Debug("failed to remove watched directory", "path", watchedPath, "err", err)
		}
	}
}

func (w *Watcher) readEvents() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			normalizedName := normalizePath(event.Name)

			// Handle newly created directories.
			if event.Has(fsnotify.Create) {
				stat, err := os.Stat(normalizedName)
				if err == nil && stat.IsDir() {
					if !w.shouldIgnoreDir(filepath.Base(normalizedName)) && !w.shouldIgnorePath(normalizedName) {
						if err := w.addDirRecursive(normalizedName); err != nil {
							slog.Error("failed to watch new directory", "path", normalizedName, "err", err)
						}
					}
					continue
				}
			}

			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				w.removeWatchedDir(normalizedName)
			}

			// Filter out irrelevant files
			if w.shouldIgnorePath(normalizedName) || w.shouldIgnoreFile(filepath.Base(normalizedName)) {
				continue
			}

			// Only care about Write, Create, Remove, Rename
			relevantOps := fsnotify.Write | fsnotify.Create | fsnotify.Remove | fsnotify.Rename
			if event.Op&relevantOps != 0 {
				event.Name = normalizedName
				w.Events <- event
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("watcher error", "err", err)
		}
	}
}

func (w *Watcher) Stop() {
	w.watcher.Close()
	close(w.Events)
}

func normalizePath(path string) string {
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}
	return filepath.Clean(path)
}

func pathEquals(left string, right string) bool {
	return strings.EqualFold(normalizePath(left), normalizePath(right))
}
