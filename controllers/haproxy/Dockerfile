FROM haproxy
# install keepalived
COPY sources.list.jessie /etc/apt/sources.list
RUN apt-get update && apt-get -y install keepalived && apt-get clean
COPY haproxy.cfg /usr/local/etc/haproxy/haproxy.cfg
COPY keepalived.conf /etc/keepalived/keepalived.conf
COPY bin/controller haproxy.tmpl keepalived.tmpl /
RUN mkdir -p /var/state/haproxy

ENTRYPOINT ["/controller"]
