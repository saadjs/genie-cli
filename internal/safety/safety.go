package safety

import (
	"regexp"
	"strings"
)

type Level int

const (
	Safe Level = iota
	Warning
	Dangerous
)

type Result struct {
	Level   Level
	Matches []string
}

var patterns = []struct {
	re    *regexp.Regexp
	level Level
	label string
}{
	// Dangerous
	{regexp.MustCompile(`\brm\b[^\n]*\s/(?:\s|$)`), Dangerous, "rm on root"},
	{regexp.MustCompile(`\bmkfs\b`), Dangerous, "mkfs"},
	{regexp.MustCompile(`\bdd\s+if=`), Dangerous, "dd if="},

	// Warning
	{regexp.MustCompile(`\brm\b[^\n]*\s-[a-zA-Z]*r[a-zA-Z]*`), Warning, "recursive rm"},
	{regexp.MustCompile(`\bsudo\b`), Warning, "sudo"},
	{regexp.MustCompile(`\bchmod\s+777\b`), Warning, "chmod 777"},
	{regexp.MustCompile(`\bkill\s+-9\b`), Warning, "kill -9"},
	{regexp.MustCompile(`\bshutdown\b`), Warning, "shutdown"},
	{regexp.MustCompile(`\breboot\b`), Warning, "reboot"},
}

func Check(command string) Result {
	r := Result{Level: Safe}
	for _, p := range patterns {
		if p.re.MatchString(command) {
			r.Matches = append(r.Matches, p.label)
			if p.level > r.Level {
				r.Level = p.level
			}
		}
	}
	if writesToDevice(command) {
		r.Matches = append(r.Matches, "write to device")
		if Dangerous > r.Level {
			r.Level = Dangerous
		}
	}
	return r
}

var deviceWritePattern = regexp.MustCompile(`(?:^|[;&|])\s*[^>\n]*>\s*(/dev/\S+)`)

func writesToDevice(command string) bool {
	matches := deviceWritePattern.FindAllStringSubmatch(command, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		target := strings.TrimRight(match[1], ";|&")
		if target != "/dev/null" {
			return true
		}
	}
	return false
}
