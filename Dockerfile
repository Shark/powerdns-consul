FROM powerdns/pdns-auth-45:latest
MAINTAINER Felix Seidel <felix@seidel.me>

ADD powerdns-consul /usr/local/bin/
ADD config.json /etc/powerdns-consul/config.json

ADD pdns.powerdns-consul.conf /etc/powerdns/pdns.conf
