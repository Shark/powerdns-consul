package consul

import (
  "strings"
  "github.com/hashicorp/consul/api"
)

func filterKVPairs(pairs []*api.KVPair, numSegments int) ([]*api.KVPair) {
  var resultPairs []*api.KVPair

  for _, pair := range pairs {
    keyTokens := strings.Split(pair.Key, "/")

    if len(keyTokens) == numSegments {
      resultPairs = append(resultPairs, pair)
    }
  }

  return resultPairs
}
