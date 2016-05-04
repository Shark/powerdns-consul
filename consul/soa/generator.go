package soa

import (
  "fmt"
  "encoding/json"
  "strconv"
  "time"
  "github.com/hashicorp/consul/api"
  "github.com/Shark/powerdns-consul/consul/iface"
)

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

type Generator struct {
  currentTime time.Time
}

func NewGenerator(currentTime time.Time) (*Generator) {
  return &Generator{currentTime}
}

func (g *Generator) RetrieveOrCreateSOAEntry(kv iface.KVStore, zone string, hostname string, hostmasterEmailAddress string, defaultTTL uint32) (entry *iface.Entry, err error) {
  prefix := fmt.Sprintf("zones/%s", zone)
  _, meta, err := kv.List(prefix, nil)

  if err != nil {
    return nil, err
  }

  lastModifyIndex := meta.LastIndex

  key := fmt.Sprintf("soa/%s", zone)
  soaEntryPair, _, err := kv.Get(key, nil)

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
      snDate := getDateFormatted(g.currentTime)

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
        // TODO: what about newSnVersion > 99?
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

      _, err = kv.Put(&api.KVPair{Key: key, Value: json}, nil)
      if err != nil {
        return nil, err
      }
    } // else nothing to do
  } else {
    // generate a new _SOA entry
    snDate := getDateFormatted(g.currentTime)
    var snVersion uint32 = 1

    if err != nil {
      return nil, err
    }

    sn := formatSoaSn(snDate, snVersion)
    soa = soaEntry{hostname, hostmasterEmailAddress, sn, 1200, 180, 1209600, int32(defaultTTL), lastModifyIndex, snDate, snVersion}

    json, err := json.Marshal(soa)

    if err != nil {
      return nil, err
    }

    _, err = kv.Put(&api.KVPair{Key: key, Value: json}, nil)
    if err != nil {
      return nil, err
    }
  }

  soaAsEntry := formatSoaEntry(&soa, defaultTTL)
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

func formatSoaEntry(sEntry *soaEntry, ttl uint32) (*iface.Entry) {
  value := fmt.Sprintf("%s %s %d %d %d %d %d", sEntry.NameServer, sEntry.EmailAddr, sEntry.Sn, sEntry.Refresh, sEntry.Retry, sEntry.Expiry, sEntry.Nx)

  return &iface.Entry{"SOA", ttl, value}
}

func getDateFormatted(time time.Time) (int) {
  formattedMonthString := fmt.Sprintf("%02d", time.Month())
  formattedDayString := fmt.Sprintf("%02d", time.Day())

  dateString := fmt.Sprintf("%d%s%s", time.Year(), formattedMonthString, formattedDayString)
  date, err := strconv.Atoi(dateString)

  if err != nil {
    return 0
  }

  return date
}
