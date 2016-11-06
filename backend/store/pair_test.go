package store

import (
	"testing"
)

var normalizeKeyTests = []struct {
	in       string
	expected string
}{
	{"a/b", "a/b"},
	{"/a/b", "a/b"},
	{"/a/b/", "a/b"},
	{"//a/b//", "/a/b/"},
}

func TestNormalizeKey(t *testing.T) {
	for _, tt := range normalizeKeyTests {
		actual := normalizeKey(tt.in)
		if actual != tt.expected {
			t.Errorf("normalizeKey(%s): expected %s, actual %s", tt.in, tt.expected, actual)
		}
	}
}
