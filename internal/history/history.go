package history

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type Entry struct {
	Timestamp string `json:"timestamp"`
	Request   string `json:"request"`
	Command   string `json:"command"`
}

// PathOverride allows tests to redirect history storage.
var PathOverride string

func path() string {
	if PathOverride != "" {
		return PathOverride
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".genie_history")
}

// Save appends an entry. The file is private to the user since requests can
// mention paths, hosts, and other details.
func Save(request, command string) error {
	p := path()
	if p == "" {
		return nil
	}

	data, err := json.Marshal(Entry{
		Timestamp: time.Now().Format(time.RFC3339),
		Request:   request,
		Command:   command,
	})
	if err != nil {
		return err
	}

	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_ = f.Chmod(0o600) // tighten files created by older versions
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Load returns the n most recent entries, newest first.
func Load(n int) ([]Entry, error) {
	p := path()
	if p == "" || n <= 0 {
		return nil, nil
	}

	f, err := os.Open(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Keep only the last n entries in a ring so memory stays bounded no
	// matter how large the file grows.
	ring := make([]Entry, n)
	count := 0
	// bufio.Reader, not Scanner: one oversized entry must not break history.
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		var e Entry
		if len(line) > 0 && json.Unmarshal(line, &e) == nil {
			ring[count%n] = e
			count++
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	size := min(count, n)
	entries := make([]Entry, size)
	for i := range size {
		entries[i] = ring[(count-1-i)%n]
	}
	return entries, nil
}
