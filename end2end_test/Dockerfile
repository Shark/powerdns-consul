FROM ubuntu:xenial
MAINTAINER Felix Seidel <felix@seidel.me>

RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get -y install pdns-server pdns-backend-pipe dnsutils sudo ca-certificates curl unzip

RUN consul_version="0.6.4" && \
    cd /tmp && \
    curl -o consul.zip "https://releases.hashicorp.com/consul/${consul_version}/consul_${consul_version}_linux_amd64.zip" && \
    unzip consul.zip -d /usr/local/bin && \
    rm consul.zip && \
    addgroup --system consul && \
    adduser --system --no-create-home --disabled-login --ingroup consul consul && \
    mkdir -p /var/local/lib/consul && \
    chown consul:consul /var/local/lib/consul && \
    mkdir /etc/consul /etc/powerdns-consul

RUN apt-get -y autoremove --purge unzip && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ADD pdns.powerdns-consul.conf /etc/powerdns/pdns.d/
ADD config.json /etc/powerdns-consul/config.json
ADD consul.json /etc/consul/
ADD end2end_test /usr/local/bin/

ADD powerdns-consul /usr/local/bin/
CMD ["/bin/bash", "/usr/local/bin/end2end_test"]
