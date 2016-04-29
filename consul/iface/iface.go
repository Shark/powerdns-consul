package iface

type Query struct {
  Name string
  Type string
}

type Entry struct {
  Type string
  Ttl uint32
  Payload string
}
