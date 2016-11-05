consul_kv_set() {
  declare key="$1"
  declare value="$2"
  curl -s -f -X PUT -d "$value" "http://127.0.0.1:8500/v1/kv/$key" > /dev/null
}

consul_kv_prepare() {
  consul_kv_set "zones/example.com/A" '[{"Payload": "127.0.0.1"}]'
  consul_kv_set "zones/example.com/MX" '[{"Payload": "10\tmx1.example.com"},{"Payload": "20\tmx2.example.com"}]'
  consul_kv_set "zones/example.com/mx1/A" '[{"Payload": "127.0.0.2"}]'
  consul_kv_set "zones/example.com/mx2/A" '[{"Payload": "127.0.0.3"}]'
}

test_prepare() {
  unzip /kv/consul.zip -d /usr/local/bin
  mkdir -p /etc/consul /var/local/lib/consul

  cp /test/consul/consul.json /etc/consul/
  /usr/local/bin/consul agent -config-dir=/etc/consul &
  cp /test/consul/config.json /etc/powerdns-consul/config.json

  sleep 5

  consul_kv_prepare
}
