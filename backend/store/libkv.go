package store

import (
	"log"

	"github.com/docker/libkv"
	libkvStore "github.com/docker/libkv/store"
	"github.com/docker/libkv/store/consul"
	"github.com/docker/libkv/store/etcd"
)

func NewLibKVStore(kvBackend string, kvAddress []string) Store {
	consul.Register()
	etcd.Register()

	client, err := libkv.NewStore(
		libkvStore.Backend(kvBackend),
		kvAddress,
		nil,
	)

	if err != nil {
		log.Panicf("Unable to instantiate libkv client: %v", err)
	}

	return &LibKVStore{client}
}

type LibKVStore struct {
	upstream libkvStore.Store
}

func (s LibKVStore) Get(key string) (Pair, error) {
	pair, err := s.upstream.Get(key)

	if err != nil {
		return nil, err
	}

	return &PairImpl{pair.Key, pair.Value, pair.LastIndex}, nil
}

func (s LibKVStore) Put(key string, value []byte, options *WriteOptions) error {
	return s.upstream.Put(key, value, nil)
}

func (s LibKVStore) List(directory string) (result []Pair, err error) {
	pairs, err := s.upstream.List(directory)

	if err != nil {
		return nil, err
	}

	result = make([]Pair, len(pairs))

	for i, pair := range pairs {
		result[i] = &PairImpl{pair.Key, pair.Value, pair.LastIndex}
	}

	return result, nil
}

func (s LibKVStore) AtomicPut(key string, value []byte, previous Pair, options *WriteOptions) (bool, Pair, error) {
	var prev *libkvStore.KVPair
	if previous != nil {
		prev = &libkvStore.KVPair{previous.Key(), previous.Value(), previous.LastIndex()}
	}

	ok, pair, err := s.upstream.AtomicPut(key, value, prev, nil)

	if err != nil {
		return false, nil, err
	}

	return ok, &PairImpl{pair.Key, pair.Value, pair.LastIndex}, err
}
