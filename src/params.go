package main

import (
	"os"
	"flag"
	"path/filepath"
	log "github.com/Sirupsen/logrus"
)

func check(e error, m string) {
    if e != nil {
		log.Error("[Error]: ", m , e)
	}
}

type Params struct {
		influxurl string
		influxdb string
		influxuser string
		influxpass string
		geoipdb string
		format string
		limit int
		files []string
		refresh int
		daemon	bool
		poll	bool
}

func (p *Params) init() {
	var file_path string
	var err error

	flag.StringVar(&p.format, "format", "influx", "Output format. influx | json")
	flag.StringVar(&p.influxurl, "influxurl", "http://localhost:8086", "Influx url connection")
	flag.StringVar(&p.influxdb, "influxdb", "", "Influx db name")
	flag.StringVar(&p.influxuser, "influxuser", "", "Influx username")
	flag.StringVar(&p.influxpass, "influxpass", "", "Influx password")
	flag.StringVar(&file_path, "filepath", "/var/log/nginx/access.log", "Log files to analyze, wildcard allowed between quotes.")
	flag.StringVar(&p.geoipdb, "geoipdb", "GeoLite2-City.mmdb", "Geoip db file.")
	flag.BoolVar(&p.daemon,"daemon", false, "Run in daemon mode. Tail files and send metrics continuously by limit or by refresh")
	flag.BoolVar(&p.poll,"poll", false, "Use poll instead of inotify. daemon mode")
	flag.IntVar(&p.limit, "limit", 2000, "Limit batch size")
	flag.IntVar(&p.refresh, "refresh", 120, "Send metrics every refresh seconds. daemon mode")

	flag.Parse()

	p.files , err = filepath.Glob(file_path)
	if err != nil {
		log.Fatal(err)
	}

	p.checkParams()
}

func (p *Params) checkParams() {
	if p.format != "influx" && p.format != "json"{
		flag.Usage()
		log.Info("Check your format params. influx | json ")
		os.Exit(1) 
	}
	if p.format == "influx" {
		if ( len(p.influxdb) == 0 || len(p.influxurl) == 0 ) { 
			flag.Usage()
			log.Info("Check your influxdb and/or influxurl params.")
			os.Exit(1) 
		}
	}
}