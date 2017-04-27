[![](https://images.microbadger.com/badges/image/rawmind/rancher-catalog-stats.svg)](https://microbadger.com/images/rawmind/rancher-catalog-stats "Get your own image badge on microbadger.com")

rancher-catalog-stats
=====================

This image run rancher-catalog-stats app. It comes from [rawmind/alpine-base][alpine-base].

## Build

```
docker build -t rawmind/rancher-catalog-stats:<version> .
```

## Versions

- `0.0.1` [(Dockerfile)](https://github.com/rawmind0/rancher-catalog-stats/blob/0.0.1/Dockerfile)


## Usage

This image run rancher-catalog-stats service. Rancher-catalog-stats get metrics from rancher nginx logs files and send them to a influx in order to be explored by a grafana. It will get and send metrics every refresh seconds. 

```
Usage of rancher-catalog-stats:
  -filepath string
      Log files to analyze, wildcard allowed between quotes. (default "/var/log/nginx/access.log")
  -format string
      Output format. influx | json (default "influx")
  -geoipdb string
      Geoip db file. (default "GeoLite2-City.mmdb")
  -influxdb string
      Influx db name
  -influxpass string
      Influx password
  -influxurl string
      Influx url connection (default "http://localhost:8086")
  -influxuser string
      Influx username
  -limit int
      Limit batch size (default 2000)
  -refresh int
      Get metrics every refresh seconds (default 120)
```

NOTE: You need influx already installed and running. The influx db would be created if doesn't exist.

[alpine-base]: https://github.com/rawmind0/alpine-base
