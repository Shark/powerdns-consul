package soa

import (
  "reflect"
  "testing"
  "time"
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

var retrieveOrCreateSOAEntryTests = []struct {
  zone string
  hostname string
  hostmasterEmailAddress string
  defaultTTL uint32
  lastModifyIndex uint64
  existingSoaEntry *api.KVPair
  expectedEntry *iface.Entry
}{
  {"example.com", "ns.example.com.", "hostmaster.example.com.", 3600, 0, nil, &iface.Entry{"SOA", 3600, "ns.example.com. hostmaster.example.com. 2016050400 1200 180 1209600 3600"}},
  {"example.com", "ns.example.com.", "hostmaster.example.com.", 3600, 2342, &api.KVPair{Value: []byte("{\"SnModifyIndex\":2342,\"SnDate\":20160504,\"SnVersion\":1}")}, &iface.Entry{"SOA", 3600, "ns.example.com. hostmaster.example.com. 2016050401 1200 180 1209600 3600"}},
  {"example.com", "ns.example.com.", "hostmaster.example.com.", 3600, 2343, &api.KVPair{Value: []byte("{\"SnModifyIndex\":2342,\"SnDate\":20160504,\"SnVersion\":1}")}, &iface.Entry{"SOA", 3600, "ns.example.com. hostmaster.example.com. 2016050402 1200 180 1209600 3600"}},
}

func TestRetrieveOrCreateSOAEntry(t *testing.T) {
  for _, tt := range retrieveOrCreateSOAEntryTests {
    listFunc := func(prefix string, q *api.QueryOptions) (api.KVPairs, *api.QueryMeta, error) {
      return nil, &api.QueryMeta{LastIndex: tt.lastModifyIndex}, nil
    }

    getFunc := func(key string, q *api.QueryOptions) (*api.KVPair, *api.QueryMeta, error) {
      return tt.existingSoaEntry, nil, nil
    }

    putFunc := func(p *api.KVPair, q *api.WriteOptions) (*api.WriteMeta, error) {
      return nil, nil
    }

    kv := &MockKVStore{listFunc: listFunc, getFunc: getFunc, putFunc: putFunc}
    time, _ := time.Parse("2006-01-02", "2016-05-04")
    cfg := &GeneratorConfig{"ns.example.com.", "hostmaster.example.com.", 1200, 180, 1209600, 3600}
    generator := NewGenerator(cfg, time)
    actual, err := generator.RetrieveOrCreateSOAEntry(kv, tt.zone, tt.hostname, tt.hostmasterEmailAddress, tt.defaultTTL)

    if err != nil {
      t.Errorf("TestRetrieveOrCreateSOAEntry: unexpected error %v", err)
    }

    if !reflect.DeepEqual(actual, tt.expectedEntry) {
      t.Errorf("TestRetrieveOrCreateSOAEntry: actual %v, expected %v", actual, tt.expectedEntry)
    }
  }
}

func TestFormatSoaSn(t *testing.T) {
  actual := formatSoaSn(20160504, 01)

  if actual != 2016050401 {
    t.Errorf("TestFormatSoaSn: actual %v, expected %v", actual, 2016050401)
  }
}

func TestFormatSoaEntry(t *testing.T) {
  soaEntry := &soaEntry{"A", "B", 1, 2, 3, 4, 5}
  actual := formatSoaEntry(soaEntry, 6)
  expected := &iface.Entry{"SOA", 6, "A B 1 2 3 4 5"}

  if !reflect.DeepEqual(actual, expected) {
    t.Errorf("TestFormatSoaEntry: actual %v, expected %v", actual, expected)
  }
}

func TestGetDateFormatted(t *testing.T) {
  time, _ := time.Parse("2006-01-02", "2016-05-04")
  actual := getDateFormatted(time)

  if actual != 20160504 {
    t.Errorf("TestGetDateFormatted: actual %d, expected %d", actual, 20160504)
  }
}
