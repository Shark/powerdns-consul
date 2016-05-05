package consul

import (
  "reflect"
  "testing"
  "github.com/hashicorp/consul/api"
  "github.com/Shark/powerdns-consul/consul/iface"
)

type MockKVStore struct {
  getFunc func(key string, q *api.QueryOptions) (*api.KVPair, *api.QueryMeta, error)
  putFunc func(p *api.KVPair, q *api.WriteOptions) (*api.WriteMeta, error)
  keysFunc func(prefix string, separator string, q *api.QueryOptions) ([]string, *api.QueryMeta, error)
  listFunc func(prefix string, q *api.QueryOptions) (api.KVPairs, *api.QueryMeta, error)
}

func (kv *MockKVStore) Get(key string, q *api.QueryOptions) (*api.KVPair, *api.QueryMeta, error) {
  return kv.getFunc(key, q)
}

func (kv *MockKVStore) Put(p *api.KVPair, q *api.WriteOptions) (*api.WriteMeta, error) {
  return kv.putFunc(p, q)
}

func (kv *MockKVStore) Keys(prefix string, separator string, q *api.QueryOptions) ([]string, *api.QueryMeta, error) {
  return kv.keysFunc(prefix, separator, q)
}

func (kv *MockKVStore) List(prefix string, q *api.QueryOptions) (api.KVPairs, *api.QueryMeta, error) {
  return kv.listFunc(prefix, q)
}

func TestAllZones(t *testing.T) {
  keysFunc := func(prefix string, separator string, q *api.QueryOptions) ([]string, *api.QueryMeta, error) {
    return []string{"zones/a/", "zones/b/", "zones/c/", "zones/d", "zones", ""}, nil, nil
  }
  kv := &MockKVStore{keysFunc: keysFunc}
  expected := []string{"a", "b", "c"}
  actual, err := allZones(kv)

  if err != nil {
    t.Errorf("TestAllZones: unexpected error %v", err)
  }

  if !reflect.DeepEqual(expected, actual) {
    t.Errorf("TestAllZones: expected %v, actual %v", expected, actual)
  }
}

var findZoneTests = []struct {
  zones []string
  name string
  expectedZone string
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
  entries []*api.KVPair
  zone string
  remainder string
  expectedKVPairs []*api.KVPair
}{
  {
    []*api.KVPair{
      &api.KVPair{Key: "zones/example.com/A", Value: []byte("Value")},
      &api.KVPair{Key: "zones/example.com/TXT", Value: []byte("Value")},
      &api.KVPair{Key: "zones/example.com/sub/A", Value: []byte("NoValue")},
      &api.KVPair{Key: "zones/example.com", Value: []byte("NoValue")},
      &api.KVPair{Key: "some/other", Value: []byte("NoValue")},
    },
    "example.com",
    "",
    []*api.KVPair{
      &api.KVPair{Key: "zones/example.com/A", Value: []byte("Value")},
      &api.KVPair{Key: "zones/example.com/TXT", Value: []byte("Value")},
    },
  },
  {
    []*api.KVPair{
      &api.KVPair{Key: "zones/example.com/sub/A", Value: []byte("Value")},
      &api.KVPair{Key: "zones/example.com/sub/TXT", Value: []byte("Value")},
      &api.KVPair{Key: "zones/example.com/A", Value: []byte("NoValue")},
      &api.KVPair{Key: "zones/example.com/sub", Value: []byte("NoValue")},
      &api.KVPair{Key: "some/other", Value: []byte("NoValue")},
    },
    "example.com",
    "sub",
    []*api.KVPair{
      &api.KVPair{Key: "zones/example.com/sub/A", Value: []byte("Value")},
      &api.KVPair{Key: "zones/example.com/sub/TXT", Value: []byte("Value")},
    },
  },
}

func TestFindKVPairsForZone(t *testing.T) {
  for _, tt := range findKVPairsForZoneTests {
    listFunc := func(prefix string, q *api.QueryOptions) (api.KVPairs, *api.QueryMeta, error) {
      return tt.entries, nil, nil
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
  entries []*api.KVPair
  zone string
  remainder string
  filterEntryType string
  defaultTTL uint32
  expectedEntries []*iface.Entry
}{
  {
    []*api.KVPair{
      &api.KVPair{Key: "zones/example.com/A", Value: []byte("[{\"Payload\":\"Value\"}]")},
      &api.KVPair{Key: "zones/example.com/TXT", Value: []byte("[{\"TTL\":3600,\"Payload\":\"SomeOtherValue\"}]")},
      &api.KVPair{Key: "zones/example.com/MX", Value: []byte("[{\"Payload\":\"10\\tmx1.example.com\"},{\"Payload\":\"20\\tmx2.example.com\"}]")},
      &api.KVPair{Key: "zones/example.com/CNAME", Value: []byte("invalid_json")},
      &api.KVPair{Key: "zones/example.com/sub/A", Value: []byte("NoValue")},
      &api.KVPair{Key: "zones/example.com", Value: []byte("[{\"TTL\":3600,\"Payload\":\"SomeOtherValue\"}]")},
      &api.KVPair{Key: "some/other", Value: []byte("NoValue")},
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
    []*api.KVPair{
      &api.KVPair{Key: "zones/example.com/sub/A", Value: []byte("[{\"Payload\":\"Value\"}]")},
      &api.KVPair{Key: "zones/example.com/sub/TXT", Value: []byte("[{\"TTL\":3600,\"Payload\":\"SomeOtherValue\"}]")},
      &api.KVPair{Key: "zones/example.com/sub/MX", Value: []byte("[{\"Payload\":\"10\\tmx1.example.com\"},{\"Payload\":\"20\\tmx2.example.com\"}]")},
      &api.KVPair{Key: "zones/example.com/sub/CNAME", Value: []byte("invalid_json")},
      &api.KVPair{Key: "zones/example.com/A", Value: []byte("NoValue")},
      &api.KVPair{Key: "zones/example.com/sub", Value: []byte("[{\"TTL\":3600,\"Payload\":\"SomeOtherValue\"}]")},
      &api.KVPair{Key: "some/other", Value: []byte("NoValue")},
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
    []*api.KVPair{
      &api.KVPair{Key: "zones/example.com/A", Value: []byte("invalid_json")},
      &api.KVPair{Key: "zones/example.com/sub/A", Value: []byte("NoValue")},
      &api.KVPair{Key: "some/other", Value: []byte("NoValue")},
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
    listFunc := func(prefix string, q *api.QueryOptions) (api.KVPairs, *api.QueryMeta, error) {
      return tt.entries, nil, nil
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
