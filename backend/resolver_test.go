package backend

import (
	"reflect"
	"sort"
	"testing"

	"github.com/Shark/powerdns-consul/backend/iface"
	"github.com/docker/libkv/store"
)

type MockKVStore struct {
	getFunc       func(string) (*store.KVPair, error)
	putFunc       func(key string, value []byte, options *store.WriteOptions) error
	listFunc      func(directory string) ([]*store.KVPair, error)
	atomicPutFunc func(key string, value []byte, previous *store.KVPair, options *store.WriteOptions) (bool, *store.KVPair, error)
}

func (kv *MockKVStore) Get(key string) (*store.KVPair, error) {
	return kv.getFunc(key)
}

func (kv *MockKVStore) Put(key string, value []byte, options *store.WriteOptions) error {
	return kv.putFunc(key, value, options)
}

func (kv *MockKVStore) List(directory string) ([]*store.KVPair, error) {
	return kv.listFunc(directory)
}

func (kv *MockKVStore) AtomicPut(key string, value []byte, previous *store.KVPair, options *store.WriteOptions) (bool, *store.KVPair, error) {
	return kv.atomicPutFunc(key, value, previous, options)
}

func TestAllZones(t *testing.T) {
	listFunc := func(directory string) ([]*store.KVPair, error) {
		return []*store.KVPair{
			&store.KVPair{Key: "zones/a/"},
			&store.KVPair{Key: "zones/b/A"},
			&store.KVPair{Key: "zones/c/sub/A"},
			&store.KVPair{Key: "zones/d"},
			&store.KVPair{Key: "zones"},
			&store.KVPair{Key: ""},
		}, nil
	}
	kv := &MockKVStore{listFunc: listFunc}
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
	entries         []*store.KVPair
	zone            string
	remainder       string
	expectedKVPairs []*store.KVPair
}{
	{
		[]*store.KVPair{
			&store.KVPair{Key: "zones/example.com/A", Value: []byte("Value")},
			&store.KVPair{Key: "zones/example.com/TXT", Value: []byte("Value")},
			&store.KVPair{Key: "zones/example.com/sub/A", Value: []byte("NoValue")},
			&store.KVPair{Key: "zones/example.com", Value: []byte("NoValue")},
			&store.KVPair{Key: "some/other", Value: []byte("NoValue")},
		},
		"example.com",
		"",
		[]*store.KVPair{
			&store.KVPair{Key: "zones/example.com/A", Value: []byte("Value")},
			&store.KVPair{Key: "zones/example.com/TXT", Value: []byte("Value")},
		},
	},
	{
		[]*store.KVPair{
			&store.KVPair{Key: "zones/example.com/sub/A", Value: []byte("Value")},
			&store.KVPair{Key: "zones/example.com/sub/TXT", Value: []byte("Value")},
			&store.KVPair{Key: "zones/example.com/A", Value: []byte("NoValue")},
			&store.KVPair{Key: "zones/example.com/sub", Value: []byte("NoValue")},
			&store.KVPair{Key: "some/other", Value: []byte("NoValue")},
		},
		"example.com",
		"sub",
		[]*store.KVPair{
			&store.KVPair{Key: "zones/example.com/sub/A", Value: []byte("Value")},
			&store.KVPair{Key: "zones/example.com/sub/TXT", Value: []byte("Value")},
		},
	},
}

func TestFindKVPairsForZone(t *testing.T) {
	for _, tt := range findKVPairsForZoneTests {
		listFunc := func(directory string) ([]*store.KVPair, error) {
			return tt.entries, nil
		}
		kv := &MockKVStore{listFunc: listFunc}
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
	entries         []*store.KVPair
	zone            string
	remainder       string
	filterEntryType string
	defaultTTL      uint32
	expectedEntries []*iface.Entry
}{
	{
		[]*store.KVPair{
			&store.KVPair{Key: "zones/example.com/A", Value: []byte("[{\"Payload\":\"Value\"}]")},
			&store.KVPair{Key: "zones/example.com/TXT", Value: []byte("[{\"TTL\":3600,\"Payload\":\"SomeOtherValue\"}]")},
			&store.KVPair{Key: "zones/example.com/MX", Value: []byte("[{\"Payload\":\"10\\tmx1.example.com\"},{\"Payload\":\"20\\tmx2.example.com\"}]")},
			&store.KVPair{Key: "zones/example.com/CNAME", Value: []byte("invalid_json")},
			&store.KVPair{Key: "zones/example.com/sub/A", Value: []byte("NoValue")},
			&store.KVPair{Key: "zones/example.com", Value: []byte("[{\"TTL\":3600,\"Payload\":\"SomeOtherValue\"}]")},
			&store.KVPair{Key: "some/other", Value: []byte("NoValue")},
		},
		"example.com",
		"",
		"ANY",
		60,
		[]*iface.Entry{
			&iface.Entry{"A", 60, "Value"},
			&iface.Entry{"TXT", 3600, "SomeOtherValue"},
			&iface.Entry{"MX", 60, "10\tmx1.example.com"},
			&iface.Entry{"MX", 60, "20\tmx2.example.com"},
		},
	},
	{
		[]*store.KVPair{
			&store.KVPair{Key: "zones/example.com/sub/A", Value: []byte("[{\"Payload\":\"Value\"}]")},
			&store.KVPair{Key: "zones/example.com/sub/TXT", Value: []byte("[{\"TTL\":3600,\"Payload\":\"SomeOtherValue\"}]")},
			&store.KVPair{Key: "zones/example.com/sub/MX", Value: []byte("[{\"Payload\":\"10\\tmx1.example.com\"},{\"Payload\":\"20\\tmx2.example.com\"}]")},
			&store.KVPair{Key: "zones/example.com/sub/CNAME", Value: []byte("invalid_json")},
			&store.KVPair{Key: "zones/example.com/A", Value: []byte("NoValue")},
			&store.KVPair{Key: "zones/example.com/sub", Value: []byte("[{\"TTL\":3600,\"Payload\":\"SomeOtherValue\"}]")},
			&store.KVPair{Key: "some/other", Value: []byte("NoValue")},
		},
		"example.com",
		"sub",
		"ANY",
		60,
		[]*iface.Entry{
			&iface.Entry{"A", 60, "Value"},
			&iface.Entry{"TXT", 3600, "SomeOtherValue"},
			&iface.Entry{"MX", 60, "10\tmx1.example.com"},
			&iface.Entry{"MX", 60, "20\tmx2.example.com"},
		},
	},
	{
		[]*store.KVPair{
			&store.KVPair{Key: "zones/example.com/A", Value: []byte("invalid_json")},
			&store.KVPair{Key: "zones/example.com/sub/A", Value: []byte("NoValue")},
			&store.KVPair{Key: "some/other", Value: []byte("NoValue")},
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
		listFunc := func(directory string) ([]*store.KVPair, error) {
			return tt.entries, nil
		}
		kv := &MockKVStore{listFunc: listFunc}
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
