FROM scratch

ADD dnsreg /dnsreg

ENV PIPE_PATH /var/run/dnsreg/ips
ENV ETCD_URL http://etcd1.isd.ictu:4001
ENV ETCD_BASE_KEY /skydns

ENTRYPOINT ["/dnsreg"]
