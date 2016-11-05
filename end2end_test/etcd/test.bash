test_prepare() {
  cd /kv
  tar xzvf etcd.tar.gz

  cd "etcd-v${ETCD_VERSION}-linux-amd64"
  ./etcd &
  cp /test/etcd/config.json /etc/powerdns-consul/config.json

  sleep 2

  etcd_kv_prepare
}

etcd_kv_prepare() {
  ./etcdctl mk "zones/example.com/A" '[{"Payload": "127.0.0.1"}]'
  ./etcdctl mk "zones/example.com/MX" '[{"Payload": "10\tmx1.example.com"},{"Payload": "20\tmx2.example.com"}]'
  ./etcdctl mk "zones/example.com/mx1/A" '[{"Payload": "127.0.0.2"}]'
  ./etcdctl mk "zones/example.com/mx2/A" '[{"Payload": "127.0.0.3"}]'
}
