package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hpcloud/tail"
	_ "github.com/influxdata/influxdb1-client"
	influx "github.com/influxdata/influxdb1-client/v2"
	"github.com/oschwald/maxminddb-golang"
	log "github.com/sirupsen/logrus"
)

type reqLocation struct {
	City    string `json:"city"`
	Country struct {
		Name    string
		ISOCode string
	} `json:"country"`
}

type Request struct {
	Ip        string      `json:"ip"`        // Remote IP address of the client
	Proto     string      `json:"proto"`     // HTTP protocol
	Method    string      `json:"method"`    // Request method (GET, POST, etc)
	Host      string      `json:"host"`      // Requested hostname
	Path      string      `json:"path"`      // Requested path
	Status    string      `json:"status"`    // Responses status code (200, 400, etc)
	Referer   string      `json:"referer"`   // Referer (usually is set to "-")
	Agent     string      `json:"agent"`     // User agent string
	Uid       string      `json:"uid"`       // User agent string
	Location  reqLocation `json:"location"`  // Remote IP location
	Timestamp time.Time   `json:"timestamp"` // Request timestamp (UTC)
}

// Parse nginx request data
// Example: "GET http://foobar.com/ HTTP/1.1"
func (req *Request) parseRequest(str string) error {
	chunks := strings.Split(str, " ")
	if len(chunks) != 3 {
		return fmt.Errorf("invalid request format")
	}

	req.Method = chunks[0]
	req.Path = chunks[1]
	req.Proto = chunks[2]

	return nil
}

// Parse nginx log timestamp
// Example: 21/Mar/2016:02:33:29 +0000
func (req *Request) parseTimestamp(str string) error {
	ts, err := time.Parse("02/Jan/2006:15:04:05 -0700", str)
	if err == nil {
		req.Timestamp = ts
	}
	return err
}

func (r *Request) getPoint() *influx.Point {
	var n = "requests"
	v := map[string]interface{}{
		"ip":  r.Ip,
		"uid": r.Uid,
	}
	t := map[string]string{
		"host":            r.Host,
		"ip":              r.Ip,
		"uid":             r.Uid,
		"method":          r.Method,
		"path":            r.Path,
		"status":          r.Status,
		"city":            r.Location.City,
		"country":         r.Location.Country.Name,
		"country_isocode": r.Location.Country.ISOCode,
	}

	m, err := influx.NewPoint(n, t, v, r.Timestamp)
	if err != nil {
		log.Warn(err)
	}

	return m

}

func (r *Request) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func (r *Request) printJson() {
	j, err := json.Marshal(r)
	if err != nil {
		log.Error("json")
	}
	fmt.Println(string(j))

}

func (r *Request) printInflux() {
	p := r.getPoint()
	fmt.Println(p.String())
}

func (r *Request) getLocation(geoipdb string) {
	db, err := maxminddb.Open(geoipdb)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ip := net.ParseIP(r.Ip)

	var record struct {
		City struct {
			Names map[string]string `maxminddb:"names"`
		} `maxminddb:"city"`
		Country struct {
			Names   map[string]string `maxminddb:"names"`
			ISOCode string            `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	} // Or any appropriate struct

	err = db.Lookup(ip, &record)
	if err != nil {
		log.Warnf("[WARN] error looking up geolocation for ip %s: %v", ip, err)
		r.printJson()
		return
	}

	r.Location.City = record.City.Names["en"]
	r.Location.Country.Name = record.Country.Names["en"]
	r.Location.Country.ISOCode = record.Country.ISOCode
}

// Get data from the input string
func (r *Request) getData(str string, geoipdb string) error {
	var cli_ip string

	// Log format V2 with Cloudfare info
	//        log_format main '[$time_local] $http_host $remote_addr $http_x_forwarded_for, $proxy_address '
	//                        '"$request" $status $body_bytes_sent "$http_referer" '
	//                        '"$http_user_agent" $request_time $upstream_response_time "$http_x_install_uuid"';
	logFormatV2 := "^\\[([^\\]]+)\\] ([^ ]+) ([^ ]+) ([^ ]+), ([^ ]+) \"([^\"]*)\" ([^ ]+) ([^ ]+) \"([^\"]*)\" \"([^\"]*)\" ([^ ]+) ([^ ]+) \"([^\"]*)\""

	// Log format V1 direct connection
	//        log_format main '[$time_local] $http_host $remote_addr $http_x_forwarded_for '
	//                        '"$request" $status $body_bytes_sent "$http_referer" '
	//                        '"$http_user_agent" $request_time $upstream_response_time "$http_x_install_uuid"';
	logFormatV1 := "^\\[([^\\]]+)\\] ([^ ]+) ([^ ]+) ([^ ]+) \"([^\"]*)\" ([^ ]+) ([^ ]+) \"([^\"]*)\" \"([^\"]*)\" ([^ ]+) ([^ ]+) \"([^\"]*)\""

	logFormatVersion := "2"

	logline, err := regexp.Compile(logFormatV2)
	if err != nil {
		log.Fatal(err)
		return err
	}

	submatches := logline.FindStringSubmatch(str)

	// If log format is not V2, trying with log format V1
	if len(submatches) == 0 {
		logline, err = regexp.Compile(logFormatV1)
		if err != nil {
			log.Fatal(err)
			return err
		}
		submatches = logline.FindStringSubmatch(str)
		if len(submatches) > 0 {
			logFormatVersion = "1"
		}
	}

	if (len(submatches) != 14 && len(submatches) != 13) || submatches[2] == "-" || submatches[2] == "localhost" {
		//log.Warn(submatches)
		return errors.New("Bad format.")
	}

	if len(submatches[4]) < 7 {
		cli_ip = submatches[3]
	} else {
		cli_ip = submatches[4]
	}

	if strings.Contains(cli_ip, ",") {
		ips := strings.Split(cli_ip, ",")
		cli_ip = ips[len(ips)-1]
	}

	r.Ip = cli_ip
	r.Host = submatches[2]

	err = r.parseTimestamp(submatches[1])

	if err != nil {
		log.Error("Could not parse timestamp")
	}

	r.getLocation(geoipdb)

	if logFormatVersion == "2" {
		r.Status = submatches[7]
		r.Referer = submatches[9]
		r.Agent = submatches[10]
		r.Uid = submatches[13]

		err := r.parseRequest(submatches[6])

		if err != nil {
			log.Error("Could not parse request")
		}
	}

	if logFormatVersion == "1" {
		r.Status = submatches[6]
		r.Referer = submatches[8]
		r.Agent = submatches[9]
		r.Uid = submatches[12]

		err := r.parseRequest(submatches[5])

		if err != nil {
			log.Error("Could not parse request")
		}
	}

	return nil
}

type ChannelList struct {
	Readers map[string](chan struct{})
	Writers map[string](chan *Request)
}

func NewChannelList() *ChannelList {
	c := &ChannelList{
		Readers: map[string]chan struct{}{},
		Writers: map[string]chan *Request{},
	}
	return c
}

func (c *ChannelList) addReader(f string) chan struct{} {
	newChan := make(chan struct{}, 1)
	c.Readers[f] = newChan

	return newChan
}

func (c *ChannelList) addWriter(f string) chan *Request {
	newChan := make(chan *Request, 1)
	c.Writers[f] = newChan

	return newChan
}

func (c *ChannelList) Add(f string) (chan struct{}, chan *Request, error) {
	if len(f) == 0 {
		return nil, nil, fmt.Errorf("Channel name is nil")
	}

	return c.addReader(f), c.addWriter(f), nil
}

func (c *ChannelList) Get(f string) (chan struct{}, chan *Request, bool) {
	r, rok := c.Readers[f]
	w, wok := c.Writers[f]
	return r, w, rok && wok
}

func (c *ChannelList) Delete(k string) {
	if reader, writer, ok := c.Get(k); ok {
		close(reader)
		close(writer)
	}
	delete(c.Readers, k)
	delete(c.Writers, k)
}

func (c *ChannelList) Send(k string) {
	if c.Readers[k] != nil {
		c.Readers[k] <- struct{}{}
	}
}

func (c *ChannelList) SendAll() {
	for k := range c.Readers {
		c.Send(k)
	}
}

func (c *ChannelList) Len() int {
	return len(c.Readers)
}

type Requests struct {
	Exit    chan os.Signal
	Control *ChannelList
	Config  Params
}

func newRequests(conf Params) *Requests {
	var r = &Requests{
		Control: NewChannelList(),
		Config:  conf,
	}

	r.Exit = make(chan os.Signal, 1)
	signal.Notify(r.Exit, os.Interrupt, os.Kill)

	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true

	if conf.debug {
		log.SetLevel(log.DebugLevel)
	}

	return r
}

func (r *Requests) Close() {
	close(r.Exit)

}

func (r *Requests) sendToInflux(data chan *Request) {
	var points []influx.Point
	var index, p_len int

	i := newInflux(r.Config.influxurl, r.Config.influxdb, r.Config.influxuser, r.Config.influxpass)

	if i.Check(5) {
		stop := make(chan struct{}, 1)
		connected := i.CheckConnect(r.Config.refresh, stop)
		defer close(stop)
		defer i.Close()

		ticker := time.NewTicker(time.Second * time.Duration(r.Config.refresh))

		index = 0
		for {
			select {
			case <-connected:
				return
			case <-ticker.C:
				log.Info("Sync: Sending ", len(points), " points")
				if len(points) > 0 {
					if !i.sendToInflux(points, 1) {
						return
					}
					points = []influx.Point{}
				}
			case req, ok := <-data:
				if !ok {
					p_len = len(points)
					if p_len > 0 {
						log.Info("Finalyzing batch: Sending ", p_len, " points")
						if i.sendToInflux(points, 1) {
							points = []influx.Point{}
						}
					}
					return
				}
				p := req.getPoint()
				points = append(points, *p)
				p_len = len(points)
				if p_len == r.Config.limit {
					log.Info("Running batch: Sending ", p_len, " points")
					if !i.sendToInflux(points, 1) {
						return
					}
					points = []influx.Point{}
				}
				index++
			}
		}
	}
}

func (r *Requests) getDataByFile(f string) {
	fileInfo, err := os.Stat(f)
	if err != nil {
		log.Infof("Error accessing file %s, skipping...", f)
		return
	}
	fileModTime := fileInfo.ModTime()
	oldLimit, _ := time.ParseDuration(r.Config.filesOld)
	if time.Since(fileModTime) > oldLimit {
		log.Infof("File %s is older than %s, skipping...", f, oldLimit)
		return
	}

	t_mode := tail.Config{Follow: r.Config.daemon, ReOpen: r.Config.daemon, Poll: r.Config.poll}

	log.Info("Analyzing ", f)
	t, err := tail.TailFile(f, t_mode)
	if err != nil {
		log.Fatal(err)
	}

	stop, data, ok := r.Control.Get(f)
	if !ok {
		log.Error("Getting reader channels ", f)
		return
	}

	defer log.Info("Closed file ", f)
	defer t.Cleanup()

	ticker := time.NewTicker(time.Second * time.Duration(60))

	for {
		select {
		case <-ticker.C:
			fileInfo, err := os.Stat(f)
			if os.IsNotExist(err) {
				log.Infof("File %s not exist, closing...", f)
				t.Kill(nil)
				return
			}
			filePos, _ := t.Tell()
			if fileSize := fileInfo.Size(); fileSize > 0 {
				fileModTime := fileInfo.ModTime()
				fileProcessed := filePos * 100 / fileSize
				if fileProcessed >= 100 && time.Since(fileModTime) > oldLimit {
					log.Infof("File %s processed and older than %s, closing...", f, oldLimit)
					t.Kill(nil)
					return
				}
				log.Infof("File %s processed %d%%", f, fileProcessed)
			}
		case line := <-t.Lines:
			if line == nil {
				err = t.Stop()
				if err != nil {
					log.Error("Could not stop")
				}
				return
			}
			r.getData(string(line.Text), data)
		case <-stop:
			t.Kill(nil)
			return
		}
	}
}

func (r *Requests) getReadersByFiles(in, out *sync.WaitGroup) {
	files, err := filepath.Glob(r.Config.filesPath)
	if err != nil {
		log.Fatal(err)
	}

	newFiles := 0
	for _, f := range files {
		if _, _, ok := r.Control.Get(f); ok {
			continue
		}

		_, _, err := r.Control.Add(f)
		if err != nil {
			log.Error("Creating control channels ", f)
			continue
		}

		in.Add(1)
		go func(file string) {
			defer in.Done()
			defer r.Control.Delete(file)
			defer log.Debug("Closed reader ", file)
			r.getDataByFile(file)
		}(f)

		out.Add(1)
		go func(file string) {
			defer out.Done()
			defer log.Debug("Closed writer ", file)
			r.getOutput(file)
		}(f)
		newFiles++
	}

	log.Debug("New files to analyze ", newFiles, " of ", r.Control.Len())
}

func (r *Requests) getDataByFiles() {
	var in, out sync.WaitGroup
	indone := make(chan struct{}, 1)
	outdone := make(chan struct{}, 1)
	stopcheck := make(chan struct{}, 1)

	r.getReadersByFiles(&in, &out)

	go func() {
		in.Wait()
		if r.Config.daemon {
			close(stopcheck)
		}
		close(indone)
	}()

	go func() {
		out.Wait()
		close(outdone)
	}()

	if r.Config.daemon {
		go func() {
			defer log.Debug("Closed files scanner")
			ticker := time.NewTicker(time.Second * time.Duration(60))
			for {
				select {
				case <-ticker.C:
					log.Info("Refreshing files")
					r.getReadersByFiles(&in, &out)
				case <-stopcheck:
					return
				}
			}
		}()
	}

	for {
		select {
		case <-indone:
			<-outdone
			return
		case <-outdone:
			log.Error("Aborting...")
			go r.Control.SendAll()
			return
		case <-r.Exit:
			log.Info("Exit signal detected...Closing...")
			go r.Control.SendAll()
			log.Info("Waiting for close writers...")
			select {
			case <-outdone:
				return
			}
		}
	}
}

func (r *Requests) getData(line string, data chan *Request) {
	req := &Request{}
	err := req.getData(line, r.Config.geoipdb)
	if err != nil {
		//log.Debug("Error getting data, ", err)
		return
	}

	data <- req
}

func (r *Requests) getOutput(f string) {
	_, data, ok := r.Control.Get(f)
	if !ok {
		log.Error("Getting writer channel ", f)
		return
	}

	if r.Config.preview {
		r.print(data)
	} else {
		r.sendToInflux(data)
	}
}

func (r *Requests) print(data chan *Request) {
	for {
		select {
		case req, ok := <-data:
			if !ok {
				return
			}
			switch r.Config.format {
			case formatJson:
				req.printJson()
			case formatInflux:
				req.printInflux()
			}
		}
	}
}
