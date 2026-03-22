package history

import (
	"bufio"
	"encoding/json"
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

func Save(request, command string) error {
	p := path()
	if p == "" {
		return nil
	}

	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	entry := Entry{
		Timestamp: time.Now().Format(time.RFC3339),
		Request:   request,
		Command:   command,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = f.Write(append(data, '\n'))
	return err
}

func Load(n int) ([]Entry, error) {
	p := path()
	if p == "" {
		return nil, nil
	}

	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	if len(entries) > n {
		entries = entries[len(entries)-n:]
	}
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}
