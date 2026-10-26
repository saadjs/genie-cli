package ui

import (
	"fmt"
	"sync"
	"time"
)

// Spinner animates on stderr while work is in flight. It draws nothing when
// stderr isn't a terminal.
type Spinner struct {
	stop chan struct{}
	once sync.Once
	done sync.WaitGroup
}

// Spin starts a spinner with the given label. Call Stop when done.
func (t *Term) Spin(label string) *Spinner {
	s := &Spinner{stop: make(chan struct{})}
	if !t.ErrTTY || t.dumb {
		return s
	}

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	label = t.paint(true, label, dim)
	draw := func(i int) {
		fmt.Fprintf(t.Err, "\r  %s %s", t.paint(true, frames[i%len(frames)], cyan), label)
	}

	s.done.Add(1)
	go func() {
		defer s.done.Done()
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		fmt.Fprint(t.Err, "\x1b[?25l") // hide cursor
		draw(0)
		for i := 1; ; i++ {
			select {
			case <-s.stop:
				fmt.Fprint(t.Err, "\r\x1b[K\x1b[?25h") // clear line, show cursor
				return
			case <-ticker.C:
				draw(i)
			}
		}
	}()
	return s
}

// Stop clears the spinner. It is safe to call more than once.
func (s *Spinner) Stop() {
	s.once.Do(func() { close(s.stop) })
	s.done.Wait()
}
