package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/Shark/powerdns-consul/backend/schema"
	"github.com/Shark/powerdns-consul/backend/soa"
	"github.com/Shark/powerdns-consul/backend/store"
	"github.com/Shark/powerdns-consul/pdns"
)

type Config struct {
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

func resolveTransform(config Config, curStore store.Store, schema schema.Schema) func(*pdns.Request) ([]*pdns.Response, error) {
	return func(request *pdns.Request) (responses []*pdns.Response, err error) {
		query := &store.Query{request.Qname, request.Qtype}
		entries, err := schema.Resolve(query)

		if err != nil {
			return nil, err
		}

		hasZone, err := schema.HasZone(request.Qname)

		if err != nil {
			return nil, err
		}

		if hasZone && (query.Type == "ANY" || query.Type == "SOA") {
			generatorCfg := &soa.GeneratorConfig{
				SoaNameServer: config.Hostname,
				SoaEmailAddr:  config.HostmasterEmailAddress,
				SoaRefresh:    config.SoaRefresh,
				SoaRetry:      config.SoaRetry,
				SoaExpiry:     config.SoaExpiry,
				SoaNx:         config.SoaNx,
				DefaultTTL:    config.DefaultTTL,
			}
			generator := soa.NewGenerator(generatorCfg, time.Now())
			entry, err := generator.RetrieveOrCreateSOAEntry(curStore, request.Qname)

			if err != nil {
				return nil, err
			}

			if entry != nil {
				entries = append([]*store.Entry{entry}, entries...)
			}
		}

		responses = make([]*pdns.Response, len(entries))

		for index, entry := range entries {
			response := &pdns.Response{request.Qname, "IN", entry.Type, strconv.Itoa(int(entry.Ttl)), "1", entry.Payload}
			responses[index] = response
		}

		return responses, nil
	}
}

func debug(format string, a ...interface{}) {
	if os.Getenv("DEBUG") != "" {
		log.Printf(format, a...)
	}
}

func main() {
	log.SetOutput(os.Stderr)
	log.SetPrefix("powerdns-consul ")

	configFilePath := flag.String("config", "/etc/powerdns-consul/config.json", "path to the config file")
	flag.Parse()

	if _, err := os.Stat(*configFilePath); os.IsNotExist(err) {
		log.Fatalf("Unable to read config from %s: file does not exist", *configFilePath)
	}

	configFileContents, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		log.Fatalf("Unable to read config file from %s: %v", *configFilePath, err)
	}

	cfg := Config{DefaultTTL: 60, SoaRefresh: 1200, SoaRetry: 180, SoaExpiry: 1209600, SoaNx: 60}
	err = json.Unmarshal(configFileContents, &cfg)
	if err != nil {
		log.Fatalf("Unable to read config file from: %s: %v", *configFilePath, err)
	} else if cfg.Hostname == "" || cfg.HostmasterEmailAddress == "" || cfg.KVBackend == "" || cfg.KVAddress == "" {
		log.Fatal("Required settings Hostname, HostmasterEmailAddress, KVBackend or KVAddress not set in config file")
	} else if cfg.DefaultTTL == 0 || cfg.SoaRefresh == 0 || cfg.SoaRetry == 0 || cfg.SoaExpiry == 0 || cfg.SoaNx == 0 {
		log.Printf("At least one of DefaultTTL, SoaRefresh, SoaRetry, SoaExpiry or SoaNx is set to zero. Is this what you intended?")
	}

	kvStore := store.NewLibKVStore(cfg.KVBackend, []string{cfg.KVAddress})
	kvSchema := schema.NewFlatSchema(kvStore, cfg.DefaultTTL)

	handler := &pdns.Handler{resolveTransform(cfg, kvStore, kvSchema)}

	inChan, outChan, quitChan := make(chan []byte), make(chan []byte), make(chan bool)

	go func() {
		handler.Handle(inChan, outChan)
	}()

	go func() {
		bufReader := bufio.NewReader(os.Stdin)

		for {
			line, isPrefix, err := bufReader.ReadLine()

			if isPrefix {
				log.Printf("Got a prefixed line, returning")
				continue
			}

			if err != nil {
				if err == io.EOF {
					log.Printf("Received EOF on input, exiting")
					quitChan <- true
					break
				} else {
					log.Printf("Error reading line: %v", err)
					continue
				}
			}

			debug("In:  %s", line)
			inChan <- line
		}
	}()

	go func() {
		for {
			line := <-outChan
			debug("Out: %s", line)
			io.WriteString(os.Stdout, string(line))
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan)
	go func() {
		for {
			exit := false

			select {
			case signal := <-signalChan:
				if signal == syscall.SIGINT || signal == syscall.SIGTERM {
					log.Printf("Received signal: %v, exiting", signal)
					exit = true
				}
			case quit := <-quitChan:
				if quit {
					log.Printf("Exit requested by application, exiting")
					exit = true
				}
			}

			if exit {
				break
			}
		}

		wg.Done()
	}()

	wg.Wait()
}
