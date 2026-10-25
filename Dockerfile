# Ubuntu container server 40 Mo
FROM docker.io/library/ubuntu:26.04

LABEL Author chdeul
LABEL version oct-2026

# Run as non root
RUN groupadd myproxy && useradd -g myproxy -d /app myproxy
USER myproxy

WORKDIR /app
COPY proxy.bin /app/proxy.bin
COPY custom /app/custom
COPY resources /app/resources
COPY proxy.yml /app/proxy.yml

# or config VOLUME [ "/app/proxy.yml", "/app/custom", "/app/resources" ]
# TODO ENV vars to overwrite some useful yaml config : trace, secret, ...

RUN chmod +x /app/proxy.bin

EXPOSE 443
ENTRYPOINT ["/app/proxy.bin"]
CMD [ ]