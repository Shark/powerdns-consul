package backend

import (
	"testing"

	"github.com/Shark/powerdns-consul/backend/store"
)

var kvPairNumSegmentsTests = []struct {
	kvPair   store.Pair
	expected int
}{
	{store.NewPair("", []byte{}, 0), 1},
	{store.NewPair("abc", []byte{}, 0), 1},
	{store.NewPair("abc/def", []byte{}, 0), 2},
	{store.NewPair("abc/def/ghi", []byte{}, 0), 3},
}

func TestKvPairNumSegments(t *testing.T) {
	for _, tt := range kvPairNumSegmentsTests {
		actual := kvPairNumSegments(tt.kvPair)
		if actual != tt.expected {
			t.Errorf("kvPairNumSegments(%v): expected %d, actual %d", tt.kvPair, tt.expected, actual)
		}
	}
}

func TestFilterKVPairs(t *testing.T) {
	pairs := []store.Pair{
		store.NewPair("abc/def", []byte{}, 0),
		store.NewPair("abc/def/ghi", []byte{}, 0),
		store.NewPair("", []byte{}, 0),
	}

	actual := filterKVPairs(pairs, 2)

	if len(actual) != 1 {
		t.Errorf("filterKVPairs: expected len %d, actual %d", 1, len(actual))
	}

	first := actual[0]
	if first.Key() != "abc/def" {
		t.Errorf("filterKVPairs: expected to return %s, actual: %s", "abc/def", first.Key())
	}

	actual = filterKVPairs(pairs, 1)

	if len(actual) != 1 {
		t.Errorf("filterKVPairs: expected len %d, actual %d", 1, len(actual))
	}

	first = actual[0]
	if first.Key() != "" {
		t.Errorf("filterKVPairs: expected to return %s, actual: %s", "", first.Key())
	}

	actual = filterKVPairs(pairs, 0)

	if len(actual) != 0 {
		t.Errorf("filterKVPairs: expected len %d, actual %d", 0, len(actual))
	}
}

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
