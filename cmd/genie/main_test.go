package main

import (
	"bufio"
	"strings"
	"testing"
)

func TestReadInputAcceptsTrailingEOFWithoutNewline(t *testing.T) {
	input, err := readInput(bufio.NewReader(strings.NewReader("show me all files")))
	if err != nil {
		t.Fatalf("readInput returned error: %v", err)
	}
	if input != "show me all files" {
		t.Fatalf("readInput = %q, want %q", input, "show me all files")
	}
}

func TestReadInputTrimsNewline(t *testing.T) {
	input, err := readInput(bufio.NewReader(strings.NewReader("show me all files\n")))
	if err != nil {
		t.Fatalf("readInput returned error: %v", err)
	}
	if input != "show me all files" {
		t.Fatalf("readInput = %q, want %q", input, "show me all files")
	}
}

