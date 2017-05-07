package main

import (
    "time"
    log "github.com/Sirupsen/logrus"
    influx "github.com/influxdata/influxdb/client/v2"
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
        url:    u,
        db:     d,
        user:   us,
        pass:   pa,
    }

    a.timeout = time.Duration(10)
    return a
}

func (i *Influx) Check(retry int) bool {
    resp_time, _ , err := i.cli.Ping(i.timeout)
    if err != nil {
        log.Error("[Error]: ", err)
        log.Error("Influx disconnected...")
        connected := false
        for index := 0 ; index < retry && ! connected ; index++ {
            log.Error("Reconnecting ",index+1," of ", retry,"...")
            connected = i.Connect()
            if ! connected {
                time.Sleep(time.Duration(1) * time.Second)
            }
        }
        if err != nil {
            log.Error("Failed to connect to influx ", i.url)
            return false
        } else {
            log.Info("Influx response time: ", resp_time)
            return true
        }
    } else {
        log.Info("Influx response time: ", resp_time)
        return true
    }
}

func (i *Influx) CheckConnect(interval int) (chan bool) {
    ticker := time.NewTicker(time.Second * time.Duration(interval))

    connected := make(chan bool)

    go func(){
        running := false
        for {
            select {
            case <-ticker.C:
                if ! running {
                    running = true
                    if ! i.Check(2) { 
                        close(connected)
                        return 
                    } 
                    running = false
                }  
            } 
        }
    }()

    return connected
}

func (i *Influx) Connect() bool {
    var err  error
    log.Info("Connecting to Influx...")

    i.cli, err = influx.NewHTTPClient(influx.HTTPConfig{
        Addr:     i.url,
        Username: i.user,
        Password: i.pass,
    })

    if err != nil {
        log.Error("[Error]: ", err)
        return false
    } 

    if i.Check(0) {
        i.createDb()
        return true
    } else {
        return false
    }
}

func (i *Influx) Init() () {
    i.newBatch()
}

func (i *Influx) Close() {
    message := "Closing Influx connection..."
    err := i.cli.Close()
    check(err, message)
    log.Info(message)
}

func (i *Influx) createDb() {
    var err  error
    log.Info("Creating Influx database if not exists...")

    comm := "CREATE DATABASE "+i.db

    q := influx.NewQuery(comm, "", "")
    _, err = i.cli.Query(q)
    if err != nil {
        log.Error("[Error] ", err)
    } else {
        log.Info("Influx database ", i.db," created.")
    } 
}

func (i *Influx) newBatch() {
    var err  error
    message := "Creating Influx batch..."
    i.batch, err = influx.NewBatchPoints(influx.BatchPointsConfig{
        Database:  i.db,
        Precision: "s",
    })
    check(err, message)
    log.Info(message)

}

func (i *Influx) newPoint(m influx.Point) {
    message := "Adding point to batch..."
    fields, _ := m.Fields()
    pt, err := influx.NewPoint(m.Name(), m.Tags(), fields, m.Time())
    check(err, message)
    i.batch.AddPoint(pt)
}

func (i *Influx) newPoints(m []influx.Point) {
    log.Info("Adding ",len(m)," points to batch...")
    for index := range m {
        i.newPoint(m[index])
    }
}

func (i *Influx) Write() {
    start := time.Now()
    log.Info("Writing batch points...")

    // Write the batch
    err := i.cli.Write(i.batch)
    if err != nil {
        log.Error("[Error]: ", err)

    }

    log.Info("Time to write ",len(i.batch.Points())," points: ", float64((time.Since(start))/ time.Millisecond), "ms")
}

func (i *Influx) sendToInflux(m []influx.Point, retry int) bool {
    if i.Check(retry) {
        i.Init()
        i.newPoints(m)
        i.Write()
        return true
    } else {
        return false
    }
}






