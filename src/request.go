package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"encoding/json"
	"os"
	"os/signal"
	"sync"
	"net"
	log "github.com/Sirupsen/logrus"
	influx "github.com/influxdata/influxdb/client/v2"
	"github.com/oschwald/maxminddb-golang"
	"github.com/hpcloud/tail"
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
		//log.Warn(str)
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
	Input 			chan *Request
	Output 			chan *influx.Point
	Exit 			chan os.Signal
	Readers			[]chan struct{}
	Config 			Params
}

func newRequests(conf Params) *Requests {
	var r = &Requests{
		Readers: []chan struct{}{},
		Config:	conf,
	}

	r.Input = make(chan *Request,1)
	r.Output = make(chan *influx.Point,1)
	r.Exit = make(chan os.Signal, 1)
	signal.Notify(r.Exit, os.Interrupt, os.Kill)

	customFormatter := new(log.TextFormatter)
    customFormatter.TimestampFormat = "2006-01-02 15:04:05"
    log.SetFormatter(customFormatter)
    customFormatter.FullTimestamp = true

	return r
}

func (r *Requests) Close(){
	close(r.Input)
	close(r.Output)
	close(r.Exit)

}

func (r *Requests) sendToInflux() {
	var points []influx.Point
	var index,p_len int
	
	i := newInflux(r.Config.influxurl, r.Config.influxdb, r.Config.influxuser, r.Config.influxpass)

	if i.Connect() {
		connected := i.CheckConnect(r.Config.refresh)
		defer i.Close()

		ticker := time.NewTicker(time.Second * time.Duration(r.Config.refresh))

		index = 0
		for {
	        select {
	        case <-connected:
	        	return
	        case <-ticker.C:
	        	if len(points) > 0 {
	            	log.Info("Tick: Sending to influx ", len(points), " points")
	    			if i.sendToInflux(points,1) {
	    				points = []influx.Point{}
	    			} else {
	    				return
	    			}
	    		} else {
	    			log.Info("Tick: Nothing to send")
	    		}
	        case p := <- r.Output:
	        	if p != nil {
	        		points = append(points, *p)
	        		p_len = len(points)
	        		if p_len == r.Config.limit {
	            		log.Info("Batch: Sending to influx ", p_len, " points")
	            		if i.sendToInflux(points,1) {
	    					points = []influx.Point{}
	    				} else {
	    					return
	    				}
	            	}
	            	index++
	        	} else {
	        		p_len = len(points)
	        		if p_len > 0 {
	        			log.Info("Batch: Sending to influx ", p_len, " points")
	            		if i.sendToInflux(points,1) {
	    					points = []influx.Point{}
	    				}
	        		}
	        		return
	        	}
	        }
	    }
	} 
}

func (r *Requests) addReader() {
	chan_new := make(chan struct{}, 1)
	r.Readers = append(r.Readers, chan_new)
}

func (r *Requests) closeReaders() {
	for _, r_chan := range r.Readers {
		if r_chan != nil {
			r_chan <- struct{}{}
		}
	}
	r.Readers = nil
}

func (r *Requests) getDataByFile(f string, stop chan struct{}) {
	var t_mode tail.Config
	if r.Config.daemon {
		if r.Config.poll {
			t_mode = tail.Config{Follow: true, ReOpen: true, Poll: true}
		} else {
			t_mode = tail.Config{Follow: true, ReOpen: true, Poll: false}
		}
	} else {
		t_mode = tail.Config{Follow: false, ReOpen: false, Poll: false}
	}

	log.Info("Analyzing ", f)
	t, err := tail.TailFile(f, t_mode)
	if err != nil {
		log.Fatal(err)
	}

	defer close(stop)
	defer log.Info("Closing ", f)

	for {
        select {
        case line := <- t.Lines:
        	if line != nil {
            	r.getData(string(line.Text))
        	} else {
        		t.Stop()
        		return
        	}
        case <- stop:
        	t.Kill(nil)
            return
        }
    }
}

func (r *Requests) getDataByFiles() {
	var in, out sync.WaitGroup
	indone := make(chan struct{},1)
	outdone := make(chan struct{},1)

	i_chan := 0
	for _, f := range r.Config.files {
		r.addReader()
		in.Add(1)
		go func(file string, num int) {
			defer in.Done()
			r.getDataByFile(file, r.Readers[num])
		}(f, i_chan)

		out.Add(1)
		go func() {
			defer out.Done()
			r.getOutput()
		}()
		i_chan++
	}

	go func() {
		in.Wait()
		close(r.Input)
		close(r.Output)
		close(indone)
	}()

	go func() {
		out.Wait()
		close(outdone)
	}()

	for {
        select {
        case <- indone:
        	<- outdone
        	return
        case <- outdone:
        	log.Error("Aborting...")
        	go r.closeReaders()
        	return
        case <- r.Exit:
        	//close(r.Exit)
        	log.Info("Exit signal detected....Closing...")
        	go r.closeReaders()
        	select {
        	case <- outdone:
        		return
        	}
        }
    }
}

func (r *Requests) getDataByLines(lines []string) {
	for _, line := range lines {
		r.getData(string(line))
	}
}

func (r *Requests) getData(line string) {
	req, _ := NewRequest(line, r.Config.geoipdb)
	if req != nil {
		if r.Config.format == "json" {
			r.Input <- req
		} else {
			r.Output <- req.getPoint()
		}
	}
}

func (r *Requests) print() {
	if r.Config.format == "json" {
		r.printJson()
	} else {
		r.printInflux()
	}
}

func (r *Requests) getOutput() {
	if r.Config.format == "json" {
		r.printJson()
	} else {
		r.sendToInflux()
	}
}

func (r *Requests) printJson() {
	for {
        select {
        case req := <- r.Input:
        	if req != nil {
            	req.printJson()
        	} else {
        		return
        	}
        }
    }
}

func (r *Requests) printInflux() {
	for {
        select {
        case p := <- r.Output:
        	if p != nil {
            	fmt.Println(p.String())
        	} else {
        		return
        	}
        }
    }
}


