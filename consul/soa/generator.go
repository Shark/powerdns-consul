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
}

type soaRevision struct {
  SnModifyIndex uint64
  SnDate int
  SnVersion uint32
}

type GeneratorConfig struct {
  SoaNameServer string
  SoaEmailAddr string
  SoaRefresh int32
  SoaRetry int32
  SoaExpiry int32
  SoaNx int32
}

type Generator struct {
  cfg *GeneratorConfig
  currentTime time.Time
}

func NewGenerator(cfg *GeneratorConfig, currentTime time.Time) (*Generator) {
  return &Generator{cfg, currentTime}
}

func (g *Generator) RetrieveOrCreateSOAEntry(kv iface.KVStore, zone string, hostname string, hostmasterEmailAddress string, defaultTTL uint32) (entry *iface.Entry, err error) {
  prefix := fmt.Sprintf("zones/%s", zone)
  _, meta, err := kv.List(prefix, nil)

  if err != nil {
    return nil, err
  }

  lastModifyIndex := meta.LastIndex

  key := fmt.Sprintf("soa/%s", zone)
  revEntryPair, _, err := kv.Get(key, nil)

  if err != nil {
    return nil, err
  }

  rev := soaRevision{}

  if revEntryPair != nil { // use existing revision
    err = json.Unmarshal(revEntryPair.Value, &rev)

    if err != nil {
      return nil, err
    }

    if rev.SnModifyIndex != lastModifyIndex {
      // update the modify index
      rev.SnModifyIndex = lastModifyIndex

      curSnDate := getDateFormatted(g.currentTime)
      if rev.SnDate != curSnDate {
        rev.SnDate = curSnDate
        rev.SnVersion = 0
      } else {
        // TODO: what about SnVersion > 99?
        rev.SnVersion += 1
      }
    } // else nothing to do
  } else { // create a new revision
    rev.SnDate = getDateFormatted(g.currentTime)
    rev.SnVersion = 1
    rev.SnModifyIndex = lastModifyIndex
  }

  json, err := json.Marshal(rev)

  if err != nil {
    return nil, err
  }

  _, err = kv.Put(&api.KVPair{Key: key, Value: json}, nil)
  if err != nil {
    return nil, err
  }

  soa := &soaEntry{NameServer: g.cfg.SoaNameServer,
                   EmailAddr: g.cfg.SoaEmailAddr,
                   Sn: formatSoaSn(rev.SnDate, rev.SnVersion),
                   Refresh: g.cfg.SoaRefresh,
                   Retry: g.cfg.SoaRetry,
                   Expiry: g.cfg.SoaExpiry,
                   Nx: g.cfg.SoaNx}

  soaAsEntry := formatSoaEntry(soa, defaultTTL)
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
