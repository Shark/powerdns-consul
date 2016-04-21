FROM alpine:latest
MAINTAINER Felix Seidel <felix@seidel.me>

RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/main" > /etc/apk/repositories && \
    echo "http://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories && \
    echo "@testing http://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories && \
    apk add --update pdns@testing && \
    rm -rf /var/cache/apk/*

ADD pdns.conf /etc/powerdns/
ADD powerdns-consul.json.example /etc/powerdns-consul.json

ADD powerdns-consul /usr/local/bin/
CMD ["/usr/sbin/pdns_server", "--config-dir=/etc/powerdns"]
