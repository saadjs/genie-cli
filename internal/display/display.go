package display

import (
	"fmt"

	"github.com/fatih/color"
)

func Command(command string, willRun bool) {
	fmt.Println()
	color.New(color.FgCyan, color.Bold).Print("  $ ")
	color.New(color.FgWhite, color.Bold).Println(command)
	fmt.Println()

	if !willRun {
		color.New(color.FgHiBlack).Println("  Copy and paste to run, or use genie -r to auto-execute.")
	}
}
