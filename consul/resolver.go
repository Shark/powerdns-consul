package consul

import (
  "fmt"
  "strings"
  "encoding/json"
  "github.com/hashicorp/consul/api"
  log "github.com/golang/glog"
  "github.com/Shark/powerdns-consul/consul/iface"
  "github.com/Shark/powerdns-consul/consul/soa"
)

type Resolver struct {
  Config *ResolverConfig
  kv iface.KVStore
}

type ResolverConfig struct {
  Hostname string
  HostmasterEmailAddress string
  ConsulAddress string
  DefaultTTL uint32
}

type value struct {
  TTL *uint32
  Payload *string
}

func allZones(kv iface.KVStore) (zones []string, err error) {
  keys, _, err := kv.Keys("zones/", "/", nil)

  if err != nil {
    return nil, err
  }

  for _, key := range keys {
    tokens := strings.Split(key, "/")

    if len(tokens) != 3 {
      continue
    }

    zones = append(zones, tokens[1])
  }

  return zones, nil
}

func findZone(zones []string, name string) (zone string, remainder string) {
  // name is expected to conform to a format like "name.example.com."
  tokens := strings.Split(name, ".")

  if len(tokens) < 2 {
    return "", ""
  }

  if(tokens[len(tokens)-1] == "") {
    tokens = tokens[:len(tokens)-1]
  }

  start := len(tokens) - 2
  for start >= 0 {
    length_of_zone := len(tokens) - start
    current_zone_slice := make([]string, length_of_zone)
    j := 0
    for j < length_of_zone {
      current_zone_slice[j] = tokens[start + j]
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

func findKVPairsForZone(kv iface.KVStore, zone string, remainder string) ([]*api.KVPair, error) {
  var (
    prefix string
    numSegments int
  )

  if remainder != "" {
    prefix = fmt.Sprintf("zones/%s/%s", zone, remainder)
    numSegments = 4 // zones/example.invalid/remainder/A -> 4 segments
  } else {
    prefix = fmt.Sprintf("zones/%s", zone)
    numSegments = 3 // zones/example.invalid/A -> 3 segments
  }

  unfilteredPairs, _, err := kv.List(prefix, nil)

  if err != nil {
    return nil, err
  }

  return filterKVPairs(unfilteredPairs, numSegments), nil
}

func findZoneEntries(kv iface.KVStore, zone string, remainder string, filter_entry_type string, defaultTTL uint32) (entries []*iface.Entry, err error) {
  pairs, err := findKVPairsForZone(kv, zone, remainder)

  if err != nil {
    return nil, err
  }

  for _, pair := range pairs {
    entry_type_tokens := strings.Split(pair.Key, "/")
    entry_type := entry_type_tokens[len(entry_type_tokens)-1]

    if filter_entry_type == "ANY" || entry_type == filter_entry_type {
      values_in_entry := make([]value, 0)
      err = json.Unmarshal(pair.Value, &values_in_entry)

      if err != nil {
        log.Errorf("Discarding key %s: %v", pair.Key, err)
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
          log.Errorf("Discarding entry in key %s because payload is missing", pair.Key)
          continue
        }

        entry := &iface.Entry{entry_type, ttl, *value.Payload}
        entries = append(entries, entry)
      }
    }
  }

  return entries, nil
}

func NewResolver(config *ResolverConfig) (*Resolver) {
  client, err := api.NewClient(&api.Config{Address: config.ConsulAddress})

  if err != nil {
    panic(fmt.Sprintf("Unable to instantiate Consul client: %v", err))
  }

  return &Resolver{config, client.KV()}
}

func (cr *Resolver) Resolve(query *iface.Query) (entries []*iface.Entry, err error) {
  log.Infof("Received query: %v", query)

  zones, err := allZones(cr.kv)

  if err != nil {
    return nil, err
  }

  zone, remainder := findZone(zones, query.Name)
  log.Infof("zone: %s, remainder: %s", zone, remainder)

  if err != nil {
    return nil, err
  }

  if zone == "" {
    return make([]*iface.Entry, 0), nil
  }

  entries, err = findZoneEntries(cr.kv, zone, remainder, query.Type, cr.Config.DefaultTTL)

  if err != nil {
    return nil, err
  }

  if remainder == "" && (query.Type == "ANY" || query.Type == "SOA") {
    entry, err := soa.RetrieveOrCreateSOAEntry(cr.kv, zone, cr.Config.Hostname, cr.Config.HostmasterEmailAddress, cr.Config.DefaultTTL)

    if err != nil {
      return nil, err
    }

    entries = append([]*iface.Entry{entry}, entries...)
  }

  log.Infof("got %d entries", len(entries))

  return entries, nil
}
