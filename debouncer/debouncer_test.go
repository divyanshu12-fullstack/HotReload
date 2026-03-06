package debouncer

import (
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestDebouncerCollapseEvents(t *testing.T) {
	d := New(50 * time.Millisecond)
	go d.Start()

	// Send 5 rapid events
	for i := 0; i < 5; i++ {
		d.Input <- fsnotify.Event{Name: "test.go"}
		time.Sleep(10 * time.Millisecond)
	}

	select {
	case <-d.Output:
		// success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected one event, got none")
	}

	// Make sure no more events are coming
	select {
	case <-d.Output:
		t.Fatal("expected exactly one event, got multiple")
	case <-time.After(50 * time.Millisecond):
		// success
	}
}

func TestDebouncerTwoDistinctEvents(t *testing.T) {
	d := New(50 * time.Millisecond)
	go d.Start()

	d.Input <- fsnotify.Event{Name: "test1.go"}

	select {
	case <-d.Output:
		// success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected first event")
	}

	// wait longer than debouncer delay
	time.Sleep(20 * time.Millisecond)

	d.Input <- fsnotify.Event{Name: "test2.go"}

	select {
	case <-d.Output:
		// success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected second event")
	}
}
