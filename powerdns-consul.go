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
	Schemas                []SchemaConfig
	DefaultTTL             uint32
	SoaRefresh             int32
	SoaRetry               int32
	SoaExpiry              int32
	SoaNx                  int32
}

type SchemaConfig struct {
	Name      string
	KVBackend string
	KVAddress string
}

func resolveTransform(config Config, schemas []schema.Schema) func(*pdns.Request) ([]*pdns.Response, error) {
	return func(request *pdns.Request) (responses []*pdns.Response, err error) {
		query := &store.Query{request.Qname, request.Qtype}
		var entries []*store.Entry

		for _, schema := range schemas {
			schemaEntries, err := schema.Resolve(query)

			if err != nil {
				log.Printf("Schema could not resolve %v: %v", query, err)
				continue
			}

			entries = append(entries, schemaEntries...)
		}

		if query.Type == "ANY" || query.Type == "SOA" {
			for _, schema := range schemas {
				hasZone, hasZoneErr := schema.HasZone(request.Qname)

				if hasZoneErr != nil {
					log.Printf("Schema could not tell if it has zone %s: %v", request.Qname, hasZoneErr)
					continue
				}

				if hasZone {
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
					entry, err := generator.RetrieveOrCreateSOAEntry(schema.Store(), request.Qname)

					if err != nil {
						log.Printf("Schema %v failed to generate SOA entry: %v", schema, err)
					} else {
						entries = append(entries, entry)
					}

					break
				}
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
	} else if cfg.Hostname == "" || cfg.HostmasterEmailAddress == "" {
		log.Fatal("Required settings Hostname, HostmasterEmailAddress, KVBackend or KVAddress not set in config file")
	} else if len(cfg.Schemas) == 0 {
		log.Fatal("No schemas are defined in config file")
	} else if cfg.DefaultTTL == 0 || cfg.SoaRefresh == 0 || cfg.SoaRetry == 0 || cfg.SoaExpiry == 0 || cfg.SoaNx == 0 {
		log.Printf("At least one of DefaultTTL, SoaRefresh, SoaRetry, SoaExpiry or SoaNx is set to zero. Is this what you intended?")
	}

	var schemas []schema.Schema
	for _, schemaConfig := range cfg.Schemas {
		kvStore, err := store.NewLibKVStore(schemaConfig.KVBackend, []string{schemaConfig.KVAddress})

		if err != nil {
			log.Printf("Unable to create kv store for schema %v: %v", schemaConfig, err)
			continue
		}

		curSchema, err := schema.NewSchema(schemaConfig.Name, kvStore, cfg.DefaultTTL)

		if err != nil {
			log.Printf("Unable to create schema for %v: %v", schemaConfig, err)
			continue
		}

		schemas = append(schemas, curSchema)
	}

	inChan, outChan, quitChan := make(chan []byte), make(chan []byte), make(chan bool)
	handler := &pdns.Handler{resolveTransform(cfg, schemas)}

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
