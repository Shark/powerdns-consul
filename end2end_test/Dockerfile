FROM ubuntu:xenial
MAINTAINER Felix Seidel <felix@seidel.me>

RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get -y install pdns-server pdns-backend-pipe dnsutils sudo ca-certificates curl unzip && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* && \
    mkdir /etc/powerdns-consul /test

ADD pdns.powerdns-consul.conf /etc/powerdns/pdns.d/
ADD consul /test/consul
ADD end2end_test /usr/local/bin/

ADD powerdns-consul /usr/local/bin/
ENTRYPOINT ["/bin/bash", "/usr/local/bin/end2end_test"]
