FROM ubuntu:xenial
MAINTAINER Felix Seidel <felix@seidel.me>

RUN export DEBIAN_FRONTEND=noninteractive && \
    apt-get update && \
    apt-get -y install nano wget ca-certificates && \
    echo "deb http://repo.powerdns.com/ubuntu xenial-auth-master main" > /etc/apt/sources.list.d/powerdns.list && \
    echo "Package: pdns-*\nPin: origin repo.powerdns.com\nPin-Priority: 600" > /etc/apt/preferences.d/pdns && \
    wget -qO- https://repo.powerdns.com/CBC8B383-pub.asc | apt-key add - && \
    apt-get update && \
    apt-get -y install pdns-server pdns-backend-pipe && \
    apt-get -y purge wget ca-certificates && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

ADD pdns.powerdns-consul.conf /etc/powerdns/pdns.d/
ADD powerdns-consul.json.example /etc/powerdns-consul.json

ADD powerdns-consul /usr/local/bin/
CMD ["/usr/sbin/pdns_server"]
