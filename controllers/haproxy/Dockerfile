FROM haproxy
COPY sources.list.jessie /etc/apt/sources.list
RUN apt-get update && apt-get -y install ssl-cert openssl && apt-get clean
RUN /usr/sbin/make-ssl-cert generate-default-snakeoil --force-overwrite; mkdir -p /var/state/haproxy; mkdir -p /usr/local/etc/haproxy/ssl; openssl dhparam 2048 -out /usr/local/etc/haproxy/ssl/dhparam.pem
COPY default.cfg /usr/local/etc/haproxy/haproxy.cfg
COPY bin/controller haproxy.tmpl /
ENTRYPOINT ["/controller"]
