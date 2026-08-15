package xray

import (
	"strings"
	"testing"
)

func TestConciseVersionLine(t *testing.T) {
	got := conciseVersionLine([]byte("Xray 26.7.28 (Xray, Penetrates Everything.) Custom (go1.26.0 linux/amd64)\nA unified platform"))
	if got != "Xray 26.7.28" {
		t.Fatalf("got %q", got)
	}
}

func TestConciseVersionLineBoundsUnknownBanners(t *testing.T) {
	got := conciseVersionLine([]byte(strings.Repeat("نسخه", 30)))
	if len([]rune(got)) != 64 {
		t.Fatalf("got %d runes: %q", len([]rune(got)), got)
	}
}
