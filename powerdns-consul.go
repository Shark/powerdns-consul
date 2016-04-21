package main

import (
  "os"
  "flag"
  "fmt"
  "encoding/json"
  "io/ioutil"
  "github.com/hashicorp/consul/api"
  log "github.com/golang/glog"
)

type config struct {
  Hostname string
  HostmasterEmailAddress string
  ConsulAddress string
  DefaultTTL uint32
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

  var curConfig config
  err = json.Unmarshal(configFileContents, &curConfig)
  if err != nil {
    panic(fmt.Sprintf("Unable to read config file from: %s: %v", *configFilePath, err))
  }

  log.Infof("Using Hostname: %s", curConfig.Hostname)
  log.Infof("Using HostmasterEmailAddress: %s", curConfig.HostmasterEmailAddress)
  log.Infof("Using ConsulAddress: %s", curConfig.ConsulAddress)

  client, err := api.NewClient(&api.Config{Address: curConfig.ConsulAddress})

  if err != nil {
    panic(fmt.Sprintf("Unable to instantiate consul client: %v", err))
  }

  resolver := &ConsulResolver{client, curConfig.Hostname, curConfig.HostmasterEmailAddress, curConfig.DefaultTTL}

  handler := &PowerDNSHandler{resolver.Resolve}
  handler.Handle(os.Stdin, os.Stdout)
}
