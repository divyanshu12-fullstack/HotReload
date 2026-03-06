package debouncer

import (
	"time"

	"github.com/fsnotify/fsnotify"
)

type Debouncer struct {
	Input  chan fsnotify.Event
	Output chan struct{}
	delay  time.Duration
}

func New(delay time.Duration) *Debouncer {
	return &Debouncer{
		Input:  make(chan fsnotify.Event, 100),
		Output: make(chan struct{}, 10),
		delay:  delay,
	}
}

func (d *Debouncer) Start() {
	var timer *time.Timer
	// To safely receive from timer.C in the select before it's initialized,
	// we use a nil channel initially.
	var timerC <-chan time.Time

	for {
		select {
		case event, ok := <-d.Input:
			if !ok {
				if timer != nil {
					timer.Stop()
				}
				close(d.Output)
				return
			}
			_ = event

			if timer == nil {
				timer = time.NewTimer(d.delay)
				timerC = timer.C
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(d.delay)
			}
		case <-timerC:
			d.Output <- struct{}{}
		}
	}
}
