package iface

import (
  "github.com/hashicorp/consul/api"
)

type Query struct {
  Name string
  Type string
}

type Entry struct {
  Type string
  Ttl uint32
  Payload string
}

type KVStore interface {
  Get(key string, q *api.QueryOptions) (*api.KVPair, *api.QueryMeta, error)
  Put(p *api.KVPair, q *api.WriteOptions) (*api.WriteMeta, error)
  Keys(prefix string, separator string, q *api.QueryOptions) ([]string, *api.QueryMeta, error)
  List(prefix string, q *api.QueryOptions) (api.KVPairs, *api.QueryMeta, error)
}
