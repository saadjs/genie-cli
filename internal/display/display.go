package display

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

func Command(command string, willRun bool, copied bool) {
	fmt.Println()
	color.New(color.FgCyan, color.Bold).Print("  $ ")
	color.New(color.FgWhite, color.Bold).Println(command)
	fmt.Println()

	if willRun {
		return
	}
	if copied {
		color.New(color.FgGreen).Println("  Copied to clipboard!")
	} else {
		color.New(color.FgHiBlack).Println("  Copy and paste to run, or use genie -r to auto-execute.")
	}
}

func Explanation(command, explanation string) {
	fmt.Println()
	color.New(color.FgCyan, color.Bold).Print("  $ ")
	color.New(color.FgWhite, color.Bold).Println(command)
	fmt.Println()
	for _, line := range strings.Split(explanation, "\n") {
		fmt.Printf("  %s\n", line)
	}
	fmt.Println()
}
