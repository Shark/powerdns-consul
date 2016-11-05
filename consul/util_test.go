package consul

import (
	"github.com/hashicorp/consul/api"
	"testing"
)

var kvPairNumSegmentsTests = []struct {
	kvPair   *api.KVPair
	expected int
}{
	{&api.KVPair{Key: ""}, 1},
	{&api.KVPair{Key: "abc"}, 1},
	{&api.KVPair{Key: "abc/def"}, 2},
	{&api.KVPair{Key: "abc/def/ghi"}, 3},
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
	pairs := []*api.KVPair{
		&api.KVPair{Key: "abc/def"},
		&api.KVPair{Key: "abc/def/ghi"},
		&api.KVPair{Key: ""},
	}

	actual := filterKVPairs(pairs, 2)

	if len(actual) != 1 {
		t.Errorf("filterKVPairs: expected len %d, actual %d", 1, len(actual))
	}

	first := actual[0]
	if first.Key != "abc/def" {
		t.Errorf("filterKVPairs: expected to return %s, actual: %s", "abc/def", first.Key)
	}

	actual = filterKVPairs(pairs, 1)

	if len(actual) != 1 {
		t.Errorf("filterKVPairs: expected len %d, actual %d", 1, len(actual))
	}

	first = actual[0]
	if first.Key != "" {
		t.Errorf("filterKVPairs: expected to return %s, actual: %s", "", first.Key)
	}

	actual = filterKVPairs(pairs, 0)

	if len(actual) != 0 {
		t.Errorf("filterKVPairs: expected len %d, actual %d", 0, len(actual))
	}
}
