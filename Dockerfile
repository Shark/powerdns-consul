FROM ubuntu:xenial
MAINTAINER Felix Seidel <felix@seidel.me>

RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get -y install pdns-server pdns-backend-pipe && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ADD pdns.powerdns-consul.conf /etc/powerdns/pdns.d/
ADD powerdns-consul.json.example /etc/powerdns-consul.json

ADD powerdns-consul /usr/local/bin/
CMD ["/usr/sbin/pdns_server"]
