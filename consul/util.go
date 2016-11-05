package consul

import (
	"github.com/hashicorp/consul/api"
	"strings"
)

func filterKVPairs(pairs []*api.KVPair, numSegments int) []*api.KVPair {
	var resultPairs []*api.KVPair

	for _, pair := range pairs {
		if kvPairNumSegments(pair) == numSegments {
			resultPairs = append(resultPairs, pair)
		}
	}

	return resultPairs
}

func kvPairNumSegments(pair *api.KVPair) int {
	return len(strings.Split(pair.Key, "/"))
}
