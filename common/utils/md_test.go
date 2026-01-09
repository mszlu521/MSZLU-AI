package utils

import (
	_ "embed"
	"testing"
)

//go:embed test.md
var content string

func TestSplitByHeading(t *testing.T) {
	chunks := SplitByHeading(content, "##")
	for _, chunk := range chunks {
		t.Log(chunk)
	}
}
