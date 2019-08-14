package main

import (
	"fmt"
	"time"

	_ "github.com/influxdata/influxdb1-client"
	influx "github.com/influxdata/influxdb1-client/v2"
	log "github.com/sirupsen/logrus"
)

type Influx struct {
	url     string
	db      string
	user    string
	pass    string
	cli     influx.Client
	batch   influx.BatchPoints
	timeout time.Duration
}

func newInflux(u, d, us, pa string) *Influx {
	var a = &Influx{
		url:  u,
		db:   d,
		user: us,
		pass: pa,
	}

	a.timeout = time.Duration(10)
	return a
}

func (i *Influx) Check(retry int) bool {
	connected := i.Connect()
	for index := 0; index < retry && !connected; index, connected = index+1, i.Connect() {
		if !connected {
			wait := index + 1 * 5 
			log.Error("Influx disconnected...")
			log.Error("Reconnecting ", index+1, " of ", retry, "...")
			log.Info("Waiting ", wait, " seconds before retry...")
			time.Sleep(time.Duration(wait) * time.Second)
		}
	}

	if !connected {
		log.Error("Failed to connect to influx ", i.url)
		return false
	} 

	return true
}

func (i *Influx) CheckConnect(interval int, stop chan struct{}) chan bool {
	ticker := time.NewTicker(time.Second * time.Duration(interval))

	connected := make(chan bool)

	go func() {
		running := false
		for {
			select {
			case <-ticker.C:
				if !running {
					running = true
					if !i.Check(5) {
						close(connected)
						return
					}
					running = false
				}
			case <-stop:
				return
			}
		}
	}()

	return connected
}

func (i *Influx) Connect() bool {
	var err error
	if i.cli != nil {
		resp_time, _, err := i.cli.Ping(i.timeout)
		if err != nil {
			log.Error("[Error]: ", err)
			return false
		}
		log.Debug("Influx response time: ", resp_time)
		return true
	}

	log.Debug("Connecting to Influx...")

	i.cli, err = influx.NewHTTPClient(influx.HTTPConfig{
		Addr:     i.url,
		Username: i.user,
		Password: i.pass,
	})

	if err != nil {
		log.Error("[Error]: ", err)
		return false
	}

	err = i.createDb()
	if err != nil {
		log.Error("[Error]: ", err)
		return false
	}
	return true
}

func (i *Influx) Init() {
	i.newBatch()
}

func (i *Influx) Close() {
	message := "Closing Influx connection..."
	err := i.cli.Close()
	check(err, message)
	log.Debug(message)
}

func (i *Influx) createDb() error {
	var err error
	log.Debug("Creating Influx database if not exists...")

	comm := "CREATE DATABASE " + i.db

	q := influx.NewQuery(comm, "", "")
	_, err = i.cli.Query(q)
	if err != nil {
		return fmt.Errorf("[Error] %v", err)
	}
	log.Debug("Influx database ", i.db, " created.")

	return nil
}

func (i *Influx) newBatch() {
	var err error
	message := "Creating Influx batch..."
	i.batch, err = influx.NewBatchPoints(influx.BatchPointsConfig{
		Database:  i.db,
		Precision: "s",
	})
	check(err, message)
	log.Debug(message)

}

func (i *Influx) newPoint(m influx.Point) {
	message := "Adding point to batch..."
	fields, _ := m.Fields()
	pt, err := influx.NewPoint(m.Name(), m.Tags(), fields, m.Time())
	check(err, message)
	i.batch.AddPoint(pt)
}

func (i *Influx) newPoints(m []influx.Point) {
	log.Debug("Adding ", len(m), " points to batch...")
	for index := range m {
		i.newPoint(m[index])
	}
}

func (i *Influx) Write() {
	start := time.Now()
	log.Debug("Writing batch points...")

	// Write the batch
	err := i.cli.Write(i.batch)
	if err != nil {
		log.Error("[Error]: ", err)

	}

	log.Debug("Time to write ", len(i.batch.Points()), " points: ", float64((time.Since(start))/time.Millisecond), "ms")
}

func (i *Influx) sendToInflux(m []influx.Point, retry int) bool {
	if i.Check(retry) {
		i.Init()
		i.newPoints(m)
		i.Write()
		return true
	} 
	return false
}
