package backend

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Shark/powerdns-consul/backend/soa"
	"github.com/Shark/powerdns-consul/backend/store"
)

type Resolver struct {
	Config *ResolverConfig
	kv     store.Store
}

type ResolverConfig struct {
	Hostname               string
	HostmasterEmailAddress string
	KVBackend              string
	KVAddress              string
	DefaultTTL             uint32
	SoaRefresh             int32
	SoaRetry               int32
	SoaExpiry              int32
	SoaNx                  int32
}

type value struct {
	TTL     *uint32
	Payload *string
}

func allZones(kv store.Store) (zones []string, err error) {
	// backends behavior is inconsistent:
	// say a key exists at zones/example.invalid/A
	// - consul will return a pair with key zones/example.invalid/A
	// - etcd will return a pair with key zones/example.invalid
	pairs, err := kv.List("zones")

	if err != nil {
		return nil, err
	}

	var zonesMap = make(map[string]bool)

	for _, pair := range pairs {
		tokens := strings.Split(normalizeKey(pair.Key()), "/")

		if len(tokens) < 2 {
			continue
		}

		zonesMap[tokens[1]] = true
	}

	zones = make([]string, len(zonesMap))
	i := 0
	for key := range zonesMap {
		zones[i] = key
		i++
	}

	return zones, nil
}

func findZone(zones []string, name string) (zone string, remainder string) {
	// name is expected to conform to a format like "name.example.com."
	normalizedName := strings.ToLower(name)
	tokens := strings.Split(normalizedName, ".")

	if len(tokens) < 2 {
		return "", ""
	}

	if tokens[len(tokens)-1] == "" {
		tokens = tokens[:len(tokens)-1]
	}

	start := len(tokens) - 2
	for start >= 0 {
		length_of_zone := len(tokens) - start
		current_zone_slice := make([]string, length_of_zone)
		j := 0
		for j < length_of_zone {
			current_zone_slice[j] = tokens[start+j]
			j++
		}
		start--

		current_zone := strings.Join(current_zone_slice, ".")

		for _, existing_zone := range zones {
			if current_zone == existing_zone {
				zone = existing_zone

				length_of_remainder := len(tokens) - length_of_zone
				if length_of_remainder > 0 {
					remainder_slice := tokens[0:length_of_remainder]
					var nonEmptyRemainderTokens []string
					for _, remainderToken := range remainder_slice {
						if remainderToken != "" {
							nonEmptyRemainderTokens = append(nonEmptyRemainderTokens, remainderToken)
						}
					}
					remainder = strings.Join(nonEmptyRemainderTokens, ".")
				} else {
					remainder = ""
				}
			}
		}
	}

	return zone, remainder
}

func findKVPairsForZone(kv store.Store, zone string, remainder string) ([]store.Pair, error) {
	var (
		prefix      string
		numSegments int
	)

	if remainder != "" {
		prefix = fmt.Sprintf("zones/%s/%s", zone, remainder)
		numSegments = 4 // zones/example.invalid/remainder/A -> 4 segments
	} else {
		prefix = fmt.Sprintf("zones/%s", zone)
		numSegments = 3 // zones/example.invalid/A -> 3 segments
	}

	unfilteredPairs, err := kv.List(prefix)

	if err != nil {
		return nil, err
	}

	return filterKVPairs(unfilteredPairs, numSegments), nil
}

func findZoneEntries(kv store.Store, zone string, remainder string, filter_entry_type string, defaultTTL uint32) (entries []*store.Entry, err error) {
	pairs, err := findKVPairsForZone(kv, zone, remainder)

	if err != nil {
		return nil, err
	}

	for _, pair := range pairs {
		entry_type_tokens := strings.Split(normalizeKey(pair.Key()), "/")
		entry_type := entry_type_tokens[len(entry_type_tokens)-1]

		if filter_entry_type == "ANY" || entry_type == filter_entry_type {
			values_in_entry := make([]value, 0)
			err = json.Unmarshal(pair.Value(), &values_in_entry)

			if err != nil {
				log.Printf("Discarding key %s: %v", normalizeKey(pair.Key()), err)
				continue
			}

			for _, value := range values_in_entry {
				var ttl uint32
				if value.TTL == nil {
					ttl = defaultTTL
				} else {
					ttl = *value.TTL
				}

				if value.Payload == nil {
					log.Printf("Discarding entry in key %s because payload is missing", normalizeKey(pair.Key()))
					continue
				}

				entry := &store.Entry{entry_type, ttl, *value.Payload}
				entries = append(entries, entry)
			}
		}
	}

	return entries, nil
}

func NewResolver(config *ResolverConfig, kvStore store.Store) *Resolver {
	return &Resolver{config, kvStore}
}

func (cr *Resolver) Resolve(query *store.Query) (entries []*store.Entry, err error) {
	zones, err := allZones(cr.kv)

	if err != nil {
		return nil, err
	}

	zone, remainder := findZone(zones, query.Name)

	if err != nil {
		return nil, err
	}

	if zone == "" {
		return make([]*store.Entry, 0), nil
	}

	entries, err = findZoneEntries(cr.kv, zone, remainder, query.Type, cr.Config.DefaultTTL)

	if err != nil {
		return nil, err
	}

	if remainder == "" && (query.Type == "ANY" || query.Type == "SOA") {
		generatorCfg := &soa.GeneratorConfig{SoaNameServer: cr.Config.Hostname,
			SoaEmailAddr: cr.Config.HostmasterEmailAddress,
			SoaRefresh:   cr.Config.SoaRefresh,
			SoaRetry:     cr.Config.SoaRetry,
			SoaExpiry:    cr.Config.SoaExpiry,
			SoaNx:        cr.Config.SoaNx,
			DefaultTTL:   cr.Config.DefaultTTL}
		generator := soa.NewGenerator(generatorCfg, time.Now())
		entry, err := generator.RetrieveOrCreateSOAEntry(cr.kv, zone)

		if err != nil {
			return nil, err
		}

		if entry != nil {
			entries = append([]*store.Entry{entry}, entries...)
		}
	}

	return entries, nil
}
