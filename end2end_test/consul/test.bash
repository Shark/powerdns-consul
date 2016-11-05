test_prepare() {
  local consul_version="0.6.4"

  local tmpdir
  tmpdir="$(mktemp -d)"
  trap "rm -r $tmpdir" EXIT
  curl -o "$tmpdir/consul.zip" "https://releases.hashicorp.com/consul/${consul_version}/consul_${consul_version}_linux_amd64.zip"
  unzip "$tmpdir"/consul.zip -d /usr/local/bin
  addgroup --system consul
  adduser --system --no-create-home --disabled-login --ingroup consul consul
  mkdir -p /etc/consul /var/local/lib/consul
  chown consul:consul /var/local/lib/consul

  cp /test/consul/consul.json /etc/consul/
  /usr/local/bin/consul agent -config-dir=/etc/consul &
  cp /test/consul/config.json /etc/powerdns-consul/config.json
}
