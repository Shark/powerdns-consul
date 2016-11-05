package store

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

type Store interface {
	Get(key string) (Pair, error)
	Put(key string, value []byte, options *WriteOptions) error
	List(directory string) ([]Pair, error)
	AtomicPut(key string, value []byte, previous Pair, options *WriteOptions) (bool, Pair, error)
}

type WriteOptions store.WriteOptions
type Backend store.Backend

var ErrKeyNotFound error = store.ErrKeyNotFound
