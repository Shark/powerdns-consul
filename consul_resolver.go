package main

import (
  "fmt"
  "strings"
  "errors"
  "github.com/hashicorp/consul/api"
  log "github.com/golang/glog"
)

type ConsulResolver struct {
  client  *api.Client
}

type entry struct {
  entry_type string
  value string
}

func allZones(client *api.Client) (zones []string, err error) {
  keys, _, err := client.KV().Keys("zones/", "/", nil)

  if err != nil {
    return nil, err
  }

  zones = make([]string, len(keys))

  for index, key := range keys {
    tokens := strings.Split(key, "/")

    if len(tokens) != 3 {
      panic("expected to get three tokens")
    }

    zones[index] = tokens[1]
  }

  return zones, nil
}

func findZone(zones []string, name string) (zone string, remainder string, err error) {
  // name is expected to conform to a format like "name.example.com."
  tokens := strings.Split(name, ".")

  if len(tokens) < 2 {
    return "", "", errors.New("zone must have two or more items")
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
          remainder = strings.Join(remainder_slice, ".")
        } else {
          remainder = ""
        }
      }
    }
  }

  return zone, remainder, nil
}

func findZoneEntries(client *api.Client, zone string, remainder string, filter_entry_type string) (entries []*entry, err error) {
  prefix := fmt.Sprintf("zones/%s/%s", zone, remainder)
  pairs, _, err := client.KV().List(prefix, nil)

  if err != nil {
    return nil, err
  }

  for _, pair := range pairs {
    entry_type_tokens := strings.Split(pair.Key, "/")
    entry_type := entry_type_tokens[len(entry_type_tokens)-1]

    if filter_entry_type == "ALL" || entry_type == filter_entry_type {
      entry := &entry{entry_type, string(pair.Value)}
      entries = append(entries, entry)
    }
  }

  return entries, nil
}

func (cr *ConsulResolver) Resolve(request *PdnsRequest) (responses []*PdnsResponse, err error) {
  zones, err := allZones(cr.client)

  if err != nil {
    return nil, err
  }

  zone, remainder, err := findZone(zones, request.qname)
  log.Infof("zone: %s, remainder: %s", zone, remainder)

  if err != nil {
    return nil, err
  }

  if zone == "" {
    return make([]*PdnsResponse, 0), nil
  }

  entries, err := findZoneEntries(cr.client, zone, remainder, request.qtype)

  if err != nil {
    return nil, err
  }

  log.Infof("got %d entries", len(entries))

  responses = make([]*PdnsResponse, len(entries))
  for index, entry := range entries {
    responses[index] = &PdnsResponse{request.qname, "IN", entry.entry_type, "60", "1", entry.value}
  }

  return responses, nil
}
