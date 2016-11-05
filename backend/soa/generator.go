package soa

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Shark/powerdns-consul/backend/store"
)

type soaEntry struct {
	NameServer string
	EmailAddr  string
	Sn         uint32
	Refresh    int32
	Retry      int32
	Expiry     int32
	Nx         int32
}

type soaRevision struct {
	SnModifyIndex uint64
	SnDate        int
	SnVersion     uint32
}

type GeneratorConfig struct {
	SoaNameServer string
	SoaEmailAddr  string
	SoaRefresh    int32
	SoaRetry      int32
	SoaExpiry     int32
	SoaNx         int32
	DefaultTTL    uint32
}

type Generator struct {
	cfg         *GeneratorConfig
	currentTime time.Time
}

func NewGenerator(cfg *GeneratorConfig, currentTime time.Time) *Generator {
	return &Generator{cfg, currentTime}
}

func (g *Generator) RetrieveOrCreateSOAEntry(kv store.Store, zone string) (entry *store.Entry, err error) {
	tries := 3
	for tries > 0 {
		entry, err = g.tryToRetrieveOrCreateSOAEntry(kv, zone)

		if err != nil {
			return nil, err
		}

		if entry != nil {
			return entry, err
		}

		tries--
	}

	return nil, nil
}

func (g *Generator) tryToRetrieveOrCreateSOAEntry(kv store.Store, zone string) (entry *store.Entry, err error) {
	prefix := fmt.Sprintf("zones/%s", zone)
	pairs, err := kv.List(prefix)

	if err != nil {
		return nil, err
	}

	var lastModifyIndex uint64
	for _, pair := range pairs {
		if lastModifyIndex == 0 || pair.LastIndex() > lastModifyIndex {
			lastModifyIndex = pair.LastIndex()
		}
	}

	key := fmt.Sprintf("soa/%s", zone)
	revEntryPair, err := kv.Get(key)

	if err != nil && err != store.ErrKeyNotFound {
		return nil, err
	}

	rev := soaRevision{}

	if revEntryPair != nil { // use existing revision
		err = json.Unmarshal(revEntryPair.Value(), &rev)

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
		rev.SnVersion = 0
		rev.SnModifyIndex = lastModifyIndex
	}

	json, err := json.Marshal(rev)

	if err != nil {
		return nil, err
	}

	ok, _, err := kv.AtomicPut(key, json, revEntryPair, nil)

	if err != nil || !ok {
		return nil, err
	}

	soa := &soaEntry{NameServer: g.cfg.SoaNameServer,
		EmailAddr: g.cfg.SoaEmailAddr,
		Sn:        formatSoaSn(rev.SnDate, rev.SnVersion),
		Refresh:   g.cfg.SoaRefresh,
		Retry:     g.cfg.SoaRetry,
		Expiry:    g.cfg.SoaExpiry,
		Nx:        g.cfg.SoaNx}

	soaAsEntry := formatSoaEntry(soa, g.cfg.DefaultTTL)
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

func formatSoaEntry(sEntry *soaEntry, ttl uint32) *store.Entry {
	value := fmt.Sprintf("%s %s %d %d %d %d %d", sEntry.NameServer, sEntry.EmailAddr, sEntry.Sn, sEntry.Refresh, sEntry.Retry, sEntry.Expiry, sEntry.Nx)

	return &store.Entry{"SOA", ttl, value}
}

func getDateFormatted(time time.Time) int {
	formattedMonthString := fmt.Sprintf("%02d", time.Month())
	formattedDayString := fmt.Sprintf("%02d", time.Day())

	dateString := fmt.Sprintf("%d%s%s", time.Year(), formattedMonthString, formattedDayString)
	date, err := strconv.Atoi(dateString)

	if err != nil {
		return 0
	}

	return date
}
