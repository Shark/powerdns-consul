package main

import (
  "bufio"
  "os"
  "io"
  "sync"
  "flag"
  "fmt"
  "os/signal"
  "encoding/json"
  "io/ioutil"
  "strconv"
  log "github.com/golang/glog"
  "github.com/Shark/powerdns-consul/consul"
  consulIface "github.com/Shark/powerdns-consul/consul/iface"
  "github.com/Shark/powerdns-consul/pdns"
)

func resolveTransform(resolver *consul.Resolver) (func(*pdns.Request) ([]*pdns.Response, error)) {
  return func(request *pdns.Request) (responses []*pdns.Response, err error) {
    query := &consulIface.Query{request.Qname, request.Qtype}
    entries, err := resolver.Resolve(query)

    if err != nil {
      return nil, err
    }

    responses = make([]*pdns.Response, len(entries))

    for index, entry := range entries {
      response := &pdns.Response{request.Qname, "IN", entry.Type, strconv.Itoa(int(entry.Ttl)), "1", entry.Payload}
      responses[index] = response
      log.Infof("Sending response: %v", response)
    }

    return responses, nil
  }
}

func main() {
  configFilePath := flag.String("config", "/etc/powerdns-consul/config.json", "path to the config file")
  flag.Parse()
  flag.Lookup("logtostderr").Value.Set("true")

  if _, err := os.Stat(*configFilePath); os.IsNotExist(err) {
    panic(fmt.Sprintf("Unable to read config from %s: file does not exist", *configFilePath))
  }

  configFileContents, err := ioutil.ReadFile(*configFilePath)
  if err != nil {
    panic(fmt.Sprintf("Unable to read config file from %s: %v", *configFilePath, err))
  }

  var cfg consul.ResolverConfig = consul.ResolverConfig{DefaultTTL: 60, SoaRefresh: 1200, SoaRetry: 180, SoaExpiry: 1209600, SoaNx: 60}
  err = json.Unmarshal(configFileContents, &cfg)
  if err != nil {
    panic(fmt.Sprintf("Unable to read config file from: %s: %v", *configFilePath, err))
  } else if(cfg.Hostname == "" || cfg.HostmasterEmailAddress == "" || cfg.ConsulAddress == "") {
    panic("Required settings Hostname, HostmasterEmailAddress or ConsulAddress not set in config file")
  } else if(cfg.DefaultTTL == 0 || cfg.SoaRefresh == 0 || cfg.SoaRetry == 0 || cfg.SoaExpiry == 0 || cfg.SoaNx == 0) {
    log.Errorf("At least one of DefaultTTL, SoaRefresh, SoaRetry, SoaExpiry or SoaNx is set to zero. Is this what you intended?")
  }

  resolver := consul.NewResolver(&cfg)

  handler := &pdns.Handler{resolveTransform(resolver)}

  in, out := make(chan []byte), make(chan []byte)

  go func() {
    handler.Handle(in, out)
  }()

  go func() {
    bufReader := bufio.NewReader(os.Stdin)

    for {
      line, isPrefix, err := bufReader.ReadLine()

      if isPrefix {
        log.Errorf("Got a prefixed line, returning")
        continue
      }

      if err != nil {
        log.Errorf("Error reading line: %v", err)
        continue
      }

      in <- line
    }
  }()

  go func() {
    for {
      line := <- out
      io.WriteString(os.Stdout, string(line))
    }
  }()

  var wg sync.WaitGroup
  wg.Add(1)
  signalChan := make(chan os.Signal, 1)
  signal.Notify(signalChan, os.Interrupt)
  go func() {
    for signal := range signalChan {
      log.Infof("Received signal: %v, exiting", signal)
      wg.Done()
    }
  }()
  wg.Wait()
}
