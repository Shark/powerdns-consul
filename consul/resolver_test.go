package consul

import (
  "reflect"
  "testing"
  "github.com/hashicorp/consul/api"
)

type MockKVStore struct {
  keysResponse []string
}

func (kv *MockKVStore) Get(key string, q *api.QueryOptions) (*api.KVPair, *api.QueryMeta, error) {
  return nil, nil, nil
}

func (kv *MockKVStore) Put(p *api.KVPair, q *api.WriteOptions) (*api.WriteMeta, error) {
  return nil, nil
}

func (kv *MockKVStore) Keys(prefix string, separator string, q *api.QueryOptions) ([]string, *api.QueryMeta, error) {
  return kv.keysResponse, nil, nil
}

func (kv *MockKVStore) List(prefix string, q *api.QueryOptions) (api.KVPairs, *api.QueryMeta, error) {
  return nil, nil, nil
}

func TestAllZones(t *testing.T) {
  kv := &MockKVStore{[]string{"zones/a/", "zones/b/", "zones/c/", "zones/d", "zones", ""}}
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
}

func TestFindZone(t *testing.T) {
  for _, tt := range findZoneTests {
    actualZone, actualRemainder := findZone(tt.zones, tt.name)

    if actualZone != tt.expectedZone || actualRemainder != tt.expectedRemainder {
      t.Errorf("TestFindZone: actual %s %s, expected %s %s", actualZone, actualRemainder, tt.expectedZone, tt.expectedRemainder)
    }
  }
}
