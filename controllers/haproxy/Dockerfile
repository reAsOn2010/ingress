FROM haproxy
COPY default.cfg /usr/local/etc/haproxy/haproxy.cfg
COPY bin/controller haproxy.tmpl /
RUN mkdir -p /var/state/haproxy
ENTRYPOINT ["/controller"]
