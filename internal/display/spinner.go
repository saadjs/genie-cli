package display

import (
	"fmt"
	"sync"
	"time"

	"github.com/fatih/color"
)

type Spinner struct {
	stop chan struct{}
	done sync.WaitGroup
}

func NewSpinner() *Spinner {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	msg := color.New(color.FgHiBlack).Sprint(" Thinking...")

	s := &Spinner{stop: make(chan struct{})}
	s.done.Add(1)

	go func() {
		defer s.done.Done()
		i := 0
		for {
			select {
			case <-s.stop:
				fmt.Print("\r\033[K") // clear the line
				return
			default:
				fmt.Printf("\r%s%s", color.New(color.FgCyan).Sprint(frames[i%len(frames)]), msg)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()

	return s
}

func (s *Spinner) Stop() {
	close(s.stop)
	s.done.Wait()
}
