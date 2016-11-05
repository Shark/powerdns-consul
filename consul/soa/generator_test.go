package soa

import (
	"reflect"
	"testing"
	"time"

	"github.com/Shark/powerdns-consul/consul/iface"
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

func TestRetrieveOrCreateSOAEntry(t *testing.T) {
	listFunc := func(directory string) ([]*store.KVPair, error) {
		return []*store.KVPair{
			&store.KVPair{LastIndex: 1234},
		}, nil
	}

	getFunc := func(key string) (*store.KVPair, error) {
		return &store.KVPair{Value: []byte("{\"SnModifyIndex\":2342,\"SnDate\":20160504,\"SnVersion\":1}"), LastIndex: 1234}, nil
	}

	atomicPutFunc := func(key string, value []byte, previous *store.KVPair, options *store.WriteOptions) (bool, *store.KVPair, error) {
		return true, nil, nil
	}

	kv := &MockKVStore{listFunc: listFunc, getFunc: getFunc, atomicPutFunc: atomicPutFunc}
	time, _ := time.Parse("2006-01-02", "2016-05-04")
	cfg := &GeneratorConfig{"ns.example.com.", "hostmaster.example.com.", 1200, 180, 1209600, 3600, 3600}
	generator := NewGenerator(cfg, time)

	actual, err := generator.RetrieveOrCreateSOAEntry(kv, "example.com")

	if err != nil || actual == nil {
		t.Errorf("TestRetrieveOrCreateSOAEntry: actual %v %v, expected not nil", err, actual)
	}

	kv.atomicPutFunc = func(key string, value []byte, previous *store.KVPair, options *store.WriteOptions) (bool, *store.KVPair, error) {
		return false, nil, nil
	}

	actual, err = generator.RetrieveOrCreateSOAEntry(kv, "example.com")

	if err != nil || actual != nil {
		t.Errorf("TestRetrieveOrCreateSOAEntry: actual %v %v, expected nil", err, actual)
	}

	failureCounter := 2
	kv.atomicPutFunc = func(key string, value []byte, previous *store.KVPair, options *store.WriteOptions) (bool, *store.KVPair, error) {
		if failureCounter > 0 {
			failureCounter--
			return false, nil, nil
		}
		return true, nil, nil
	}

	actual, err = generator.RetrieveOrCreateSOAEntry(kv, "example.com")

	if err != nil || actual == nil {
		t.Errorf("TestRetrieveOrCreateSOAEntry: actual %v %v, expected not nil", err, actual)
	}
}

var tryToRetrieveOrCreateSOAEntryTests = []struct {
	zone                   string
	hostname               string
	hostmasterEmailAddress string
	defaultTTL             uint32
	lastModifyIndex        uint64
	existingSoaEntry       *store.KVPair
	casResult              bool
	expectedEntry          *iface.Entry
}{
	{"example.com", "ns.example.com.", "hostmaster.example.com.", 3600, 0, nil, true, &iface.Entry{"SOA", 3600, "ns.example.com. hostmaster.example.com. 2016050400 1200 180 1209600 3600"}},
	{"example.com", "ns.example.com.", "hostmaster.example.com.", 3600, 2342, &store.KVPair{Value: []byte("{\"SnModifyIndex\":2342,\"SnDate\":20160504,\"SnVersion\":1}"), LastIndex: 1234}, true, &iface.Entry{"SOA", 3600, "ns.example.com. hostmaster.example.com. 2016050401 1200 180 1209600 3600"}},
	{"example.com", "ns.example.com.", "hostmaster.example.com.", 3600, 2343, &store.KVPair{Value: []byte("{\"SnModifyIndex\":2342,\"SnDate\":20160504,\"SnVersion\":1}"), LastIndex: 1234}, true, &iface.Entry{"SOA", 3600, "ns.example.com. hostmaster.example.com. 2016050402 1200 180 1209600 3600"}},
	{"example.com", "ns.example.com.", "hostmaster.example.com.", 3600, 2343, &store.KVPair{Value: []byte("{\"SnModifyIndex\":2342,\"SnDate\":20160504,\"SnVersion\":1}"), LastIndex: 1234}, false, nil},
}

func TestTryToRetrieveOrCreateSOAEntry(t *testing.T) {
	for _, tt := range tryToRetrieveOrCreateSOAEntryTests {
		listFunc := func(directory string) ([]*store.KVPair, error) {
			return []*store.KVPair{
				&store.KVPair{LastIndex: tt.lastModifyIndex},
			}, nil
		}

		getFunc := func(key string) (*store.KVPair, error) {
			return tt.existingSoaEntry, nil
		}

		atomicPutFunc := func(key string, value []byte, previous *store.KVPair, options *store.WriteOptions) (bool, *store.KVPair, error) {
			if tt.existingSoaEntry != nil && previous.LastIndex != tt.existingSoaEntry.LastIndex {
				t.Errorf("TestTryToRetrieveOrCreateSOAEntry: actual %d, expected %d", previous.LastIndex, tt.existingSoaEntry.LastIndex)
			}
			return tt.casResult, nil, nil
		}

		kv := &MockKVStore{listFunc: listFunc, getFunc: getFunc, atomicPutFunc: atomicPutFunc}
		time, _ := time.Parse("2006-01-02", "2016-05-04")
		cfg := &GeneratorConfig{"ns.example.com.", "hostmaster.example.com.", 1200, 180, 1209600, 3600, 3600}
		generator := NewGenerator(cfg, time)
		actual, err := generator.tryToRetrieveOrCreateSOAEntry(kv, tt.zone)

		if err != nil {
			t.Errorf("TestTryToRetrieveOrCreateSOAEntry: unexpected error %v", err)
		}

		if !reflect.DeepEqual(actual, tt.expectedEntry) {
			t.Errorf("TestTryToRetrieveOrCreateSOAEntry: actual %v, expected %v", actual, tt.expectedEntry)
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
