FROM haproxy
COPY sources.list.jessie /etc/apt/sources.list
RUN apt-get update && apt-get -y install ssl-cert && apt-get clean
COPY default.cfg /usr/local/etc/haproxy/haproxy.cfg
COPY bin/controller haproxy.tmpl /
RUN /usr/sbin/make-ssl-cert generate-default-snakeoil --force-overwrite; mkdir -p /var/state/haproxy; mkdir -p /usr/local/etc/haproxy/ssl
ENTRYPOINT ["/controller"]
