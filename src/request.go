package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"encoding/json"
	"os"
	"bufio"
	"net"
	log "github.com/Sirupsen/logrus"
	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/oschwald/maxminddb-golang"
)

type reqLocation struct {
	City          	string `json:"city"`
	Country        	string `json:"country"`
} 

type Request struct {
	Ip        string    `json:"ip"` // Remote IP address of the client
	Proto     string    `json:"proto"` // HTTP protocol
	Method    string    `json:"method"` // Request method (GET, POST, etc)
	Host      string    `json:"host"` // Requested hostname
	Path      string    `json:"path"` // Requested path
	Status    string    `json:"status"` // Responses status code (200, 400, etc)
	Referer   string    `json:"referer"` // Referer (usually is set to "-")
	Agent     string    `json:"agent"` // User agent string
	Location  reqLocation `json:"location"` // Remote IP location
	Timestamp time.Time `json:"timestamp"` // Request timestamp (UTC)
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

// Produce wire-formatted string for ingestion into influxdb
func (r *Request) printInflux() {
	p := r.getPoint()
	fmt.Println(p.String())
}

func (r *Request) getPoint() *influx.Point {
	var n = "requests"
    v := map[string]interface{}{
        "ip": r.Ip,
    }
    t := map[string]string{
    	"host":  r.Host,
        "ip": r.Ip,
        "method": r.Method,
        "path": r.Path,
        "status": r.Status,
        "city": r.Location.City,
        "country": r.Location.Country,
    }

	m, err := influx.NewPoint(n,t,v,r.Timestamp)
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

func (r *Request) getLocation(geoipdb string) {
    db, err := maxminddb.Open(geoipdb)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    ip := net.ParseIP(r.Ip)

    var record struct {
        City    struct {
            Names map[string]string `maxminddb:"names"`
        } `maxminddb:"city"`
        Country struct {
            Names map[string]string `maxminddb:"names"`
            ISOCode string `maxminddb:"iso_code"`
        } `maxminddb:"country"`
    } // Or any appropriate struct

    err = db.Lookup(ip, &record)
    if err != nil {
        log.Fatal(err)
    }
    
    r.Location.City = record.City.Names["en"] 
    r.Location.Country = record.Country.Names["en"]
}

// Initialize a new request from the input string
func NewRequest(str string, geoipdb string) (*Request, error) {
	var cli_ip string

	//        log_format main '[$time_local] $http_host $remote_addr $http_x_forwarded_for '
	//                        '"$request" $status $body_bytes_sent "$http_referer" '
	//                        '"$http_user_agent" $request_time $upstream_response_time';
	logline, err := regexp.Compile("^\\[([^\\]]+)\\] ([^ ]+) ([^ ]+) ([^ ]+) \"([^\"]*)\" ([^ ]+) ([^ ]+) \"([^\"]*)\" \"([^\"]*)\" ([^ ]+) ([^ ]+)")
	if err != nil {
		log.Fatal(err)
	}

	submatches := logline.FindStringSubmatch(str)

	if len(submatches) != 12 || submatches[2] == "-" || submatches[2] == "localhost" {
		log.Warn(str)
		return nil, nil
	}

	if len(submatches[4]) < 7 {
		cli_ip = submatches[3]
	} else {
		cli_ip = submatches[4]
	}

	req := &Request{
		Ip:      cli_ip,
		Host:    submatches[2],
		Status:  submatches[6],
		Referer: submatches[8],
		Agent:   submatches[9],
	}

	req.parseTimestamp(submatches[1])
	req.parseRequest(submatches[5])
	req.getLocation(geoipdb)

	return req, nil
}

type Requests struct {
	Reqs 			[]Request `json:"data"`
	Points 			[]influx.Point
}

func (r *Requests) getPoints() []influx.Point {
	/*for _, req := range r.Reqs {
	}*/

	return r.Points
}

func (r *Requests) printJson() {
	for _, req := range r.Reqs {
		req.printJson()
	}
}

func (r *Requests) printInflux() {
	for _, point := range r.Points {
		fmt.Println(point.String())
	}
}

func (r *Requests) sendToInflux(p Params) {
	var p_len, start, end int
	
	i := newInflux(p.influxurl, p.influxdb, p.influxuser, p.influxpass)

	i.Connect()
	defer i.Close()

	p_len = len(r.Points)
	if p.limit < p_len {
		log.Info("Sending to influx ", p_len, " points in batch size of ", p.limit)
		for seq := 0; p_len > end ; seq++ {
			start = (seq*p.limit)
			end = (((seq+1)*p.limit)-1)
			if p_len < end {
				end = p_len
			}
			log.Info("Sending to influx from  ", start, " to ", end)
			i.sendToInflux(r.Points[start:end])
		}
	} else {
		log.Info("Sending to influx ", p_len, " points", p.limit)
		i.sendToInflux(r.Points)
	}
}

func (r *Requests) getDataByFiles(files []string, geoipdb string) {
	for _, f := range files {
		r.getDataByFile(f, geoipdb)
	}
}

func (r *Requests) getDataByFile(f string, geoipdb string) {
	log.Info("Analyzing ", f)
	file, err := os.Open(f)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// 64Kb buffer should be big enough
	scanner := bufio.NewScanner(file)
		//var wg sync.WaitGroup

	for scanner.Scan() {
		line := scanner.Text()
		r.getData(string(line), geoipdb)
	}
}

func (r *Requests) getDataByLines(lines []string, geoipdb string) {
	for _, line := range lines {
		r.getData(string(line), geoipdb)
	}
}

func (r *Requests) getData(line string, geoipdb string) {
	req, _ := NewRequest(line, geoipdb)
	if req != nil {
		r.Reqs = append(r.Reqs, *req)

		p := req.getPoint()
		r.Points = append(r.Points, *p)
	}
}

func (r *Requests) print(f string) {
	if f == "json" {
		r.printJson()
	} else {
		r.printInflux()
	}
}


