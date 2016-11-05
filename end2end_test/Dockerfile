FROM ubuntu:xenial
MAINTAINER Felix Seidel <felix@seidel.me>

RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get -y install pdns-server pdns-backend-pipe dnsutils sudo ca-certificates curl unzip && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir /etc/powerdns-consul /test

ENV CONSUL_VERSION 0.7.0
ENV ETCD_VERSION 3.0.14

RUN mkdir /kv && \
    curl -sSL -o /kv/consul.zip "https://releases.hashicorp.com/consul/${CONSUL_VERSION}/consul_${CONSUL_VERSION}_linux_amd64.zip" && \
    curl -sSL -o /kv/etcd.tar.gz "https://github.com/coreos/etcd/releases/download/v${ETCD_VERSION}/etcd-v${ETCD_VERSION}-linux-amd64.tar.gz"

ADD pdns.powerdns-consul.conf /etc/powerdns/pdns.d/
ADD consul /test/consul
ADD etcd /test/etcd
ADD end2end_test /usr/local/bin/

ADD powerdns-consul /usr/local/bin/
ENTRYPOINT ["/bin/bash", "/usr/local/bin/end2end_test"]
