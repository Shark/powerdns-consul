# Flat Schema

In the flat schema the entries in the key-value store map directly to DNS records. For this, `powerdns-consul` uses two top-level prefixes:

- `zones/`: this is where the DNS records for a zone are stored.
- `soa/`: `powerdns-consul` generates SOA records dynamically. It will store some data concerning the state of the SOA record under this prefix.

## Zones

The DNS records for each zone are stored under `zones/<zone-root>` (i.e. `zones/example.invalid`). Each key with this prefix defines a DNS record:

- `zones/example.invalid/A` is an A record for `example.invalid`
- `zones/example.invalid/MX` is an MX record for `example.invalid`

Sub-zones are defined as subkeys:

- `zones/example.invalid/mx/A` is an A record for `mx.example.invalid`

The values of those keys are JSON-encoded and must have the following schema:

```
[
  {
    "payload": "127.0.0.1",
    "ttl": 3600
  }, {
    "payload": "127.0.0.2"
  }
]
```

Where `payload` is **mandatory** and `ttl` is optional.

`payload` is a string. Valid strings are IPv4/IPv6 addresses for A/AAAA records, host names for CNAME/MX records and any text for TXT records.

`payload` is an integer. It defaults to the key `DefaultTTL` in the configuration.
