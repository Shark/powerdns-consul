package main

import (
  "os"
  "flag"
  "github.com/hashicorp/consul/api"
)

var client, err = api.NewClient(api.DefaultConfig())
var resolver = &ConsulResolver{client}

func lookup(request *PdnsRequest) (responses []*PdnsResponse, err error) {
  return resolver.Resolve(request)
}

func main() {
  flag.Parse()
  flag.Lookup("logtostderr").Value.Set("true")

  handler := &PowerDNSHandler{lookup}
  handler.Handle(os.Stdin, os.Stdout)
}
