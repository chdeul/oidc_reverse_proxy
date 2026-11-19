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

# Possible ENV variables
# USED_URL         client url to access front proxy
# RESOURCES_PATH   context path to access local ressources
# TRACE            TRUE : logging trace, FALSE or not
# CERT_FILE        server https proxy certificates
# AGE_COOKIE       session or age in seconds (43200=12h)
# USER_HEADER      name of http Header of logged username
#
# OIDC_ISSUER      oidc provider configuration
# OIDC_IDENT
# OIDC_SECRET
# OIDC_AUTH_METHOD
# OIDC_UID_FIELD
# OIDC_ROOT_CA

RUN chmod +x /app/proxy.bin

EXPOSE 443
ENTRYPOINT ["/app/proxy.bin"]
CMD [ ]