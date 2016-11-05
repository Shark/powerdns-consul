package backend

import (
	"strings"

	"github.com/Shark/powerdns-consul/backend/store"
)

func filterKVPairs(pairs []store.Pair, numSegments int) []store.Pair {
	var resultPairs []store.Pair

	for _, pair := range pairs {
		if kvPairNumSegments(pair) == numSegments {
			resultPairs = append(resultPairs, pair)
		}
	}

	return resultPairs
}

func kvPairNumSegments(pair store.Pair) int {
	return len(strings.Split(normalizeKey(pair.Key()), "/"))
}

func normalizeKey(key string) string {
	return strings.TrimSuffix(strings.TrimPrefix(key, "/"), "/")
}
