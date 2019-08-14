package main

import (
	"flag"
	"os"

	log "github.com/sirupsen/logrus"
)

const (
	formatJson   = "json"
	formatInflux = "influx"
)

func check(e error, m string) {
	if e != nil {
		log.Error("[Error]: ", m, e)
	}
}

type Params struct {
	influxurl  string
	influxdb   string
	influxuser string
	influxpass string
	geoipdb    string
	format     string
	limit      int
	filesPath  string
	refresh    int
	daemon     bool
	debug      bool
	poll       bool
	preview    bool
}

func (p *Params) init() {
	flag.BoolVar(&p.debug, "debug", false, "Debug mode")
	flag.StringVar(&p.format, "format", formatInflux, "Output format. " + formatInflux + " | " + formatJson)
	flag.StringVar(&p.influxurl, "influxurl", "http://localhost:8086", "Influx url connection")
	flag.StringVar(&p.influxdb, "influxdb", "", "Influx db name")
	flag.StringVar(&p.influxuser, "influxuser", "", "Influx username")
	flag.StringVar(&p.influxpass, "influxpass", "", "Influx password")
	flag.StringVar(&p.filesPath, "filepath", "/var/log/nginx/access.log", "Log files to analyze, wildcard allowed between quotes.")
	flag.StringVar(&p.geoipdb, "geoipdb", "GeoLite2-City.mmdb", "Geoip db file.")
	flag.BoolVar(&p.daemon, "daemon", false, "Run in daemon mode. Tail files and send metrics continuously by limit or by refresh")
	flag.BoolVar(&p.poll, "poll", false, "Use poll instead of inotify. daemon mode")
	flag.BoolVar(&p.preview, "preview", false, "Print metrics to stdout")
	flag.IntVar(&p.limit, "limit", 2000, "Limit batch size")
	flag.IntVar(&p.refresh, "refresh", 120, "Send metrics every refresh seconds. daemon mode")

	flag.Parse()

	p.checkParams()
}

func (p *Params) checkParams() {
	if !p.daemon && p.poll {
		log.Warn("Setting -poll to false due to not daemon mode")
		p.poll = false
	}
	if p.format == formatJson && !p.preview {
		log.Warn("Setting -preview to true due to json format")
		p.preview = true
	}
	if p.format != formatInflux && p.format != formatJson {
		flag.Usage()
		log.Error("Check your format params, " + formatInflux + " | " + formatJson)
		os.Exit(1)
	}
	if p.format == "influx" && !p.preview {
		if len(p.influxdb) == 0 || len(p.influxurl) == 0 {
			flag.Usage()
			log.Error("Check your influxdb and/or influxurl params.")
			os.Exit(1)
		}
	}
}
