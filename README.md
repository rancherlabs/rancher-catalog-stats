# rancher-catalog-stats

The rancher-catalog-stats service gathers metrics from rancher nginx logs files and sends them to influxdb in order to be explored by grafana. 

Running in daemon mode will tail files and send metrics every refresh seconds. 

## Build

```
docker build -t rancherlabs/rancher-catalog-stats:latest .
```

## Usage

```
Usage of rancher-catalog-stats:
  -daemon
      Run in daemon mode. Tail files and send metrics continuously by limit or by refresh
  -fileold string
      Log files with modification time older than that, will be discarded (default "1h")
  -filepath string
      Log files to analyze, wildcard allowed between quotes. (default "/var/log/nginx/access.log")
  -format string
      Output format, influx | json (default "influx")
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
  -poll
      Use poll instead of inotify. daemon mode
  - preview
      Print metrics to stdout
  -refresh int
      Send metrics every refresh seconds. daemon mode (default 120)
```

NOTE: influxdb should already installed and running. The database will be created if doesn't already exist.

## Metrics

The format is as follows:

```
requests,city=Toronto,country=Canada,country_isocode=CA,host=git.rancher.io,ip=xx.xx.xx.xx,method=GET,path=/rancher-catalog.git/info/refs?service\=git-upload-pack,status=200,uid="XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXX" ip="xx.xx.xx.xx",uid="XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXX" 1491289498000000000
```

