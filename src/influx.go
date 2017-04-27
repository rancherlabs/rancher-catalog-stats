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
}

func newInflux(u, d, us, pa string) *Influx {
    var a = &Influx{
        url:    u,
        db:     d,
        user:   us,
        pass:   pa,
    }
    return a
}

func (i *Influx) Connect() () {
    var err  error
    message := "Connecting Influx connection..."
    i.cli, err = influx.NewHTTPClient(influx.HTTPConfig{
        Addr:     i.url,
        Username: i.user,
        Password: i.pass,
    })
    check(err, message)

    log.Info(message)

    i.createDb()
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
    message := "Creating Influx database if not exists..."

    comm := "CREATE DATABASE "+i.db

    q := influx.NewQuery(comm, "", "")
    _, err = i.cli.Query(q)
    check(err, message)
    log.Info(message)
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
    message := "Adding point to Influx batch..."
    fields, _ := m.Fields()
    pt, err := influx.NewPoint(m.Name(), m.Tags(), fields, m.Time())
    check(err, message)
    i.batch.AddPoint(pt)
}

func (i *Influx) newPoints(m []influx.Point) {
    message := "Adding points to Influx batch..."
    for index := range m {
        i.newPoint(m[index])
    }
    log.Info(message)
}

func (i *Influx) Write() {
    start := time.Now()
    message := "Writing Influx points..."

    // Write the batch
    err := i.cli.Write(i.batch)
    check(err, message)
    log.Info(message)

    log.Info("Time to write: ", float64((time.Since(start))/ time.Millisecond), "ms")
}

func (i *Influx) sendToInflux(m []influx.Point){
    i.Init()

    i.newPoints(m)

    i.Write()

}






