package consul

import (
	"strings"

	"github.com/docker/libkv/store"
)

func filterKVPairs(pairs []*store.KVPair, numSegments int) []*store.KVPair {
	var resultPairs []*store.KVPair

	for _, pair := range pairs {
		if kvPairNumSegments(pair) == numSegments {
			resultPairs = append(resultPairs, pair)
		}
	}

	return resultPairs
}

func kvPairNumSegments(pair *store.KVPair) int {
	return len(strings.Split(pair.Key, "/"))
}
