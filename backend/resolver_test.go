package backend

import (
	"reflect"
	"sort"
	"testing"

	"github.com/Shark/powerdns-consul/backend/store"
)

func TestAllZones(t *testing.T) {
	listFunc := func(directory string) ([]store.Pair, error) {
		return []store.Pair{
			store.NewPair("zones/a/", []byte{}, 0),
			store.NewPair("zones/b/A", []byte{}, 0),
			store.NewPair("zones/c/sub/A", []byte{}, 0),
			store.NewPair("zones/d", []byte{}, 0),
			store.NewPair("zones", []byte{}, 0),
			store.NewPair("", []byte{}, 0),
		}, nil
	}
	kv := store.MockStore{ListFunc: listFunc}
	expected := []string{"a", "b", "c", "d"}
	actual, err := allZones(kv)

	if err != nil {
		t.Errorf("TestAllZones: unexpected error %v", err)
	}

	sort.Strings(actual)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("TestAllZones: expected %v, actual %v", expected, actual)
	}
}

var findZoneTests = []struct {
	zones             []string
	name              string
	expectedZone      string
	expectedRemainder string
}{
	{[]string{"one.com", "two.com", "three.com"}, "one.com", "one.com", ""},
	{[]string{"one.com", "two.com", "three.com"}, "sub.one.com", "one.com", "sub"},
	{[]string{"one.com", "two.com", "three.com"}, "two.sub.one.com", "one.com", "two.sub"},
	{[]string{"one.com", "two.com", "three.com"}, ".sub.one.com", "one.com", "sub"},
	{[]string{"one.com", "two.com", "three.com"}, "a.....sub.one.com", "one.com", "a.sub"},
	{[]string{"one.com", "two.com", "three.com"}, ".one.com", "one.com", ""},
	{[]string{"one.com", "two.com", "three.com"}, "one.com.", "one.com", ""},
	{[]string{"one.com", "two.com", "three.com"}, "sub.three.com.", "three.com", "sub"},
	{[]string{"one.com", "two.com", "three.com"}, "sub.three.de", "", ""},
	{[]string{"one.com", "two.com", "three.com"}, "four.com", "", ""},
	{[]string{"one.com", "two.com", "three.com"}, "", "", ""},
	{[]string{"one.com", "two.com", "three.com"}, "öäaö.abc", "", ""},
	{[]string{"öäaö.abc"}, "öäaö.abc", "öäaö.abc", ""},
	{[]string{}, "öäaö.abc", "", ""},
	{[]string{"one.com"}, "SoME.oNe.CoM", "one.com", "some"},
}

func TestFindZone(t *testing.T) {
	for _, tt := range findZoneTests {
		actualZone, actualRemainder := findZone(tt.zones, tt.name)

		if actualZone != tt.expectedZone || actualRemainder != tt.expectedRemainder {
			t.Errorf("TestFindZone: actual %s %s, expected %s %s", actualZone, actualRemainder, tt.expectedZone, tt.expectedRemainder)
		}
	}
}

var findKVPairsForZoneTests = []struct {
	entries         []store.Pair
	zone            string
	remainder       string
	expectedKVPairs []store.Pair
}{
	{
		[]store.Pair{
			store.NewPair("zones/example.com/A", []byte("Value"), 0),
			store.NewPair("zones/example.com/TXT", []byte("Value"), 0),
			store.NewPair("zones/example.com/sub/A", []byte("Value"), 0),
			store.NewPair("zones/example.com", []byte("NoValue"), 0),
			store.NewPair("some/other", []byte("NoValue"), 0),
		},
		"example.com",
		"",
		[]store.Pair{
			store.NewPair("zones/example.com/A", []byte("Value"), 0),
			store.NewPair("zones/example.com/TXT", []byte("Value"), 0),
		},
	},
	{
		[]store.Pair{
			store.NewPair("zones/example.com/sub/A", []byte("Value"), 0),
			store.NewPair("zones/example.com/sub/TXT", []byte("Value"), 0),
			store.NewPair("zones/example.com/A", []byte("NoValue"), 0),
			store.NewPair("zones/example.com/sub", []byte("NoValue"), 0),
			store.NewPair("some/other", []byte("NoValue"), 0),
		},
		"example.com",
		"sub",
		[]store.Pair{
			store.NewPair("zones/example.com/sub/A", []byte("Value"), 0),
			store.NewPair("zones/example.com/sub/TXT", []byte("Value"), 0),
		},
	},
}

func TestFindKVPairsForZone(t *testing.T) {
	for _, tt := range findKVPairsForZoneTests {
		listFunc := func(directory string) ([]store.Pair, error) {
			return tt.entries, nil
		}
		kv := &store.MockStore{ListFunc: listFunc}
		actual, err := findKVPairsForZone(kv, tt.zone, tt.remainder)

		if err != nil {
			t.Errorf("TestFindKVPairsForZone: unexpected error %v", err)
		}

		if !reflect.DeepEqual(actual, tt.expectedKVPairs) {
			t.Errorf("TestFindKVPairsForZone: actual %v, expected %v", actual, tt.expectedKVPairs)
		}
	}
}

var findZoneEntriesTests = []struct {
	entries         []store.Pair
	zone            string
	remainder       string
	filterEntryType string
	defaultTTL      uint32
	expectedEntries []*store.Entry
}{
	{
		[]store.Pair{
			store.NewPair("zones/example.com/A", []byte("[{\"Payload\":\"Value\"}]"), 0),
			store.NewPair("zones/example.com/TXT", []byte("[{\"TTL\":3600,\"Payload\":\"SomeOtherValue\"}]"), 0),
			store.NewPair("zones/example.com/MX", []byte("[{\"Payload\":\"10\\tmx1.example.com\"},{\"Payload\":\"20\\tmx2.example.com\"}]"), 0),
			store.NewPair("zones/example.com/CNAME", []byte("invalid_json"), 0),
			store.NewPair("zones/example.com/sub/A", []byte("NoValue"), 0),
			store.NewPair("zones/example.com", []byte("[{\"TTL\":3600,\"Payload\":\"SomeOtherValue\"}]"), 0),
			store.NewPair("some/other", []byte("NoValue"), 0),
		},
		"example.com",
		"",
		"ANY",
		60,
		[]*store.Entry{
			&store.Entry{"A", 60, "Value"},
			&store.Entry{"TXT", 3600, "SomeOtherValue"},
			&store.Entry{"MX", 60, "10\tmx1.example.com"},
			&store.Entry{"MX", 60, "20\tmx2.example.com"},
		},
	},
	{
		[]store.Pair{
			store.NewPair("zones/example.com/sub/A", []byte("[{\"Payload\":\"Value\"}]"), 0),
			store.NewPair("zones/example.com/sub/TXT", []byte("[{\"TTL\":3600,\"Payload\":\"SomeOtherValue\"}]"), 0),
			store.NewPair("zones/example.com/sub/MX", []byte("[{\"Payload\":\"10\\tmx1.example.com\"},{\"Payload\":\"20\\tmx2.example.com\"}]"), 0),
			store.NewPair("zones/example.com/sub/CNAME", []byte("invalid_json"), 0),
			store.NewPair("zones/example.com/A", []byte("NoValue"), 0),
			store.NewPair("zones/example.com/sub", []byte("[{\"TTL\":3600,\"Payload\":\"SomeOtherValue\"}]"), 0),
			store.NewPair("some/other", []byte("NoValue"), 0),
		},
		"example.com",
		"sub",
		"ANY",
		60,
		[]*store.Entry{
			&store.Entry{"A", 60, "Value"},
			&store.Entry{"TXT", 3600, "SomeOtherValue"},
			&store.Entry{"MX", 60, "10\tmx1.example.com"},
			&store.Entry{"MX", 60, "20\tmx2.example.com"},
		},
	},
	{
		[]store.Pair{
			store.NewPair("zones/example.com/A", []byte("invalid_json"), 0),
			store.NewPair("zones/example.com/sub/A", []byte("NoValue"), 0),
			store.NewPair("some/other", []byte("NoValue"), 0),
		},
		"example.com",
		"",
		"ANY",
		60,
		nil,
	},
}

func TestFindZoneEntries(t *testing.T) {
	for _, tt := range findZoneEntriesTests {
		listFunc := func(directory string) ([]store.Pair, error) {
			return tt.entries, nil
		}
		kv := &store.MockStore{ListFunc: listFunc}
		actual, err := findZoneEntries(kv, tt.zone, tt.remainder, tt.filterEntryType, tt.defaultTTL)

		if err != nil {
			t.Errorf("TestFindZoneEntries: unexpected error %v", err)
		}

		if tt.expectedEntries != nil {
			if !reflect.DeepEqual(actual, tt.expectedEntries) {
				t.Errorf("TestFindZoneEntries: actual %v, expected %v", actual, tt.expectedEntries)
			}
		} else {
			if len(actual) != 0 {
				t.Errorf("TestFindZoneEntries: actual %d entries, expected 0", len(actual))
			}
		}
	}
}
