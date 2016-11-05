package iface

import (
	"github.com/docker/libkv/store"
)

type Query struct {
	Name string
	Type string
}

type Entry struct {
	Type    string
	Ttl     uint32
	Payload string
}

type KVStore interface {
	Get(key string) (*store.KVPair, error)
	Put(key string, value []byte, options *store.WriteOptions) error
	List(directory string) ([]*store.KVPair, error)
	AtomicPut(key string, value []byte, previous *store.KVPair, options *store.WriteOptions) (bool, *store.KVPair, error)
}
