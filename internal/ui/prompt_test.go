package ui

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestPromptAskMultilineScanner(t *testing.T) {
	var buf bytes.Buffer
	input := " 第一行 \n第二行\n\n"
	prompt := NewPrompt(strings.NewReader(input), &buf)

	lines, err := prompt.AskMultiline("描述")
	if err != nil {
		t.Fatalf("AskMultiline returned error: %v", err)
	}

	expected := []string{" 第一行 ", "第二行"}
	if !reflect.DeepEqual(lines, expected) {
		t.Fatalf("AskMultiline = %#v, want %#v", lines, expected)
	}
}

func TestPromptAskMultilineScannerEmpty(t *testing.T) {
	var buf bytes.Buffer
	input := "\n"
	prompt := NewPrompt(strings.NewReader(input), &buf)

	lines, err := prompt.AskMultiline("描述")
	if err != nil {
		t.Fatalf("AskMultiline returned error: %v", err)
	}

	if len(lines) != 0 {
		t.Fatalf("AskMultiline = %#v, want empty", lines)
	}
}
