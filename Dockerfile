FROM golang:1.12.8 AS builder
ENV SERVICE_NAME=rancher-catalog-stats
WORKDIR /go/src/github.com/rawmind0/rancher-catalog-stats/
ADD src .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ${SERVICE_NAME} .

FROM rawmind/alpine-base:3.8-1
MAINTAINER Raul Sanchez <rawmind@gmail.com>

#Set environment
ENV SERVICE_NAME=rancher-catalog-stats \
    SERVICE_HOME=/opt/rancher-catalog-stats \
    SERVICE_USER=rancher \
    SERVICE_UID=10012 \
    SERVICE_GROUP=rancher \
    SERVICE_GID=10012 \
    GOMAXPROCS=2
ENV PATH=${PATH}:${SERVICE_HOME}

WORKDIR $SERVICE_HOME
COPY --from=builder /go/src/github.com/rawmind0/rancher-catalog-stats/${SERVICE_NAME} ${SERVICE_HOME}
RUN  curl -sS http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.tar.gz | gunzip -c - | tar -xf - && \ 
    mv GeoLite2-City_*/GeoLite2-City.mmdb . && \
    rm -rf GeoLite2-City_* && \
    addgroup -g ${SERVICE_GID} ${SERVICE_GROUP} && \
    adduser -g "${SERVICE_NAME} user" -D -h ${SERVICE_HOME} -G ${SERVICE_GROUP} -s /sbin/nologin -u ${SERVICE_UID} ${SERVICE_USER} && \
    chown -R ${SERVICE_USER}:${SERVICE_GROUP} ${SERVICE_HOME}
USER $SERVICE_USER

