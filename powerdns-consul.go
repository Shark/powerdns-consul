package main

import (
  "os"
  "flag"
  "fmt"
  "encoding/json"
  "io/ioutil"
  "strconv"
  log "github.com/golang/glog"
  "github.com/Shark/powerdns-consul/consul"
  "github.com/Shark/powerdns-consul/pdns"
)

func resolveTransform(resolver *consul.Resolver) (func(*pdns.Request) ([]*pdns.Response, error)) {
  return func(request *pdns.Request) (responses []*pdns.Response, err error) {
    query := &consul.Query{request.Qname, request.Qtype}
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
  configFilePath := flag.String("config", "/etc/powerdns-consul.json", "path to the config file")
  flag.Parse()
  flag.Lookup("logtostderr").Value.Set("true")

  if _, err := os.Stat(*configFilePath); os.IsNotExist(err) {
    panic(fmt.Sprintf("Unable to read config from %s: file does not exist", *configFilePath))
  }

  configFileContents, err := ioutil.ReadFile(*configFilePath)
  if err != nil {
    panic(fmt.Sprintf("Unable to read config file from %s: %v", *configFilePath, err))
  }

  var curConfig consul.ResolverConfig
  err = json.Unmarshal(configFileContents, &curConfig)
  if err != nil {
    panic(fmt.Sprintf("Unable to read config file from: %s: %v", *configFilePath, err))
  }

  log.Infof("Using Hostname: %s", curConfig.Hostname)
  log.Infof("Using HostmasterEmailAddress: %s", curConfig.HostmasterEmailAddress)
  log.Infof("Using ConsulAddress: %s", curConfig.ConsulAddress)

  resolver := consul.NewResolver(&curConfig)

  handler := &pdns.Handler{resolveTransform(resolver)}
  handler.Handle(os.Stdin, os.Stdout)
}
