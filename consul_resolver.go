package main

import (
  "fmt"
  "strings"
  "errors"
  "encoding/json"
  "time"
  "strconv"
  "github.com/hashicorp/consul/api"
  log "github.com/golang/glog"
)

type ConsulResolver struct {
  client  *api.Client
  hostname string
  hostmasterEmailAddress string
}

type entry struct {
  entry_type string
  value string
}

type soaEntry struct {
  NameServer string
  EmailAddr string
  Sn uint32
  Refresh int32
  Retry int32
  Expiry int32
  Nx int32
  InternalSnModifyIndex uint64
  InternalSnDate int
  InternalSnVersion uint32
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

    if filter_entry_type == "ANY" || entry_type == filter_entry_type {
      entry := &entry{entry_type, string(pair.Value)}
      entries = append(entries, entry)
    }
  }

  return entries, nil
}

func getSOAEntry(client *api.Client, zone string, hostname string, hostmasterEmailAddress string) (entry *entry, err error) {
  prefix := fmt.Sprintf("zones/%s", zone)
  _, meta, err := client.KV().List(prefix, nil)

  if err != nil {
    return nil, err
  }

  lastModifyIndex := meta.LastIndex

  key := fmt.Sprintf("soa/%s", zone)
  soaEntryPair, _, err := client.KV().Get(key, nil)

  if err != nil {
    return nil, err
  }

  var soa soaEntry

  if soaEntryPair != nil {
    // update the existing _SOA entry
    err = json.Unmarshal(soaEntryPair.Value, &soa)

    if err != nil {
      return nil, err
    }

    if soa.InternalSnModifyIndex != lastModifyIndex {
      // update the modify index
      snDate := getCurrentDateFormatted()

      if err != nil {
        return nil, err
      }

      var newSnDate int
      var newSnVersion uint32
      var newSnModifyIndex = lastModifyIndex

      if soa.InternalSnDate != snDate {
        newSnDate = snDate
        newSnVersion = 0
      } else {
        newSnDate = soa.InternalSnDate
        newSnVersion = soa.InternalSnVersion + 1
      }

      soa.InternalSnDate = newSnDate
      soa.InternalSnVersion = newSnVersion
      soa.InternalSnModifyIndex = newSnModifyIndex

      soa.Sn = formatSoaSn(soa.InternalSnDate, soa.InternalSnVersion)

      json, err := json.Marshal(soa)

      if err != nil {
        return nil, err
      }

      _, err = client.KV().Put(&api.KVPair{Key: key, Value: json}, nil)
      if err != nil {
        return nil, err
      }
    } // else nothing to do
  } else {
    // generate a new _SOA entry
    snDate := getCurrentDateFormatted()
    var snVersion uint32 = 1

    if err != nil {
      return nil, err
    }

    sn := formatSoaSn(snDate, snVersion)
    soa = soaEntry{hostname, hostmasterEmailAddress, sn, 1200, 180, 1209600, 60, lastModifyIndex, snDate, snVersion}

    json, err := json.Marshal(soa)

    if err != nil {
      return nil, err
    }

    _, err = client.KV().Put(&api.KVPair{Key: key, Value: json}, nil)
    if err != nil {
      return nil, err
    }
  }

  soaAsEntry := formatSoaEntry(&soa)
  return soaAsEntry, nil
}

func formatSoaSn(snDate int, snVersion uint32) (sn uint32) {
  soaSnString := fmt.Sprintf("%d%02d", snDate, snVersion)
  soaSnInt, err := strconv.Atoi(soaSnString)

  if err != nil {
    panic("error generating SoaSn")
  }

  return uint32(soaSnInt)
}

func formatSoaEntry(sEntry *soaEntry) (*entry) {
  value := fmt.Sprintf("%s %s %d %d %d %d %d", sEntry.NameServer, sEntry.EmailAddr, sEntry.Sn, sEntry.Refresh, sEntry.Retry, sEntry.Expiry, sEntry.Nx)

  return &entry{"SOA", value}
}

func getCurrentDateFormatted() (int) {
  now := time.Now()
  formattedMonthString := fmt.Sprintf("%02d", now.Month())
  formattedDayString := fmt.Sprintf("%02d", now.Day())

  dateString := fmt.Sprintf("%d%s%s", now.Year(), formattedMonthString, formattedDayString)
  date, err := strconv.Atoi(dateString)

  if err != nil {
    return 0
  }

  return date
}

func (cr *ConsulResolver) Resolve(request *PdnsRequest) (responses []*PdnsResponse, err error) {
  log.Infof("Received request: %v", request)

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

  if remainder == "" && (request.qtype == "ANY" || request.qtype == "SOA") {
    soaEntry, err := getSOAEntry(cr.client, zone, cr.hostname, cr.hostmasterEmailAddress)

    if err != nil {
      return nil, err
    }

    entries = append([]*entry{soaEntry}, entries...)
  }

  log.Infof("got %d entries", len(entries))

  responses = make([]*PdnsResponse, len(entries))
  for index, entry := range entries {
    response := &PdnsResponse{request.qname, "IN", entry.entry_type, "60", "1", entry.value}
    responses[index] = response
    log.Infof("Sending response: %v", response)
  }

  return responses, nil
}
