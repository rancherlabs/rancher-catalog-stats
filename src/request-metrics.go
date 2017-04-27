package main

import (
	"os"
	"bufio"
	//"time"
	"sync"
	//"os/signal"
	log "github.com/Sirupsen/logrus"
)

/*
func get(p Params, wg *sync.WaitGroup) {
	wg.Add(3)
	go func() {
		defer wg.Done()
		go func() {
			defer wg.Done()
			var acc = newAccounts()
			getData(p, acc)
		}()
		go func() {
			defer wg.Done()
			var pro Projects
			getData(p, &pro)
		}()
	}()
}

func main() {
	var params Params 
	var wg sync.WaitGroup

	params.init()

	ticker := time.NewTicker(time.Second * time.Duration(params.refresh))
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, os.Kill)

	get(params, &wg)

	for {
        select {
        case <-ticker.C:
            get(params, &wg)
        case <- exit:
        	log.Info("Exit signal detected. Waiting for running jobs...")
        	wg.Wait()
        	log.Info("Done")
            return
        }
    }
}

func getData(p Params, obj RacherMetric) {
	obj.getData(p.url, p.accessKey, p.secretKey, p.admin, p.limit)

	if p.format == "influx" {
		i := newInflux(p.influxurl, p.influxdb, p.influxuser, p.influxpass)
		t := time.Now()
		i.sendToInflux(obj.getPoints(t))
	} else if p.format == "json" {
		obj.printJson()
	}
}*/

func getDataByLines(p Params, lines []string) {
	var reqs Requests
	reqs.getDataByLines(lines, p.geoipdb)
	log.Info("Metrics ")
	if p.format == "influx" {
		reqs.sendToInflux(p)
		//reqs.print(p.format)
	} else if p.format == "json" {
		reqs.print(p.format)
	}
}

func getDataByFile(p Params, f string) {

	log.Info("Analyzing ", f)
	file, err := os.Open(f)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// 64Kb buffer should be big enough
	scanner := bufio.NewScanner(file)
		//var wg sync.WaitGroup

	var lines []string
	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		if i < p.limit {
			i++
		} else {
			getDataByLines(p, lines)
			lines = lines[:0]
			i = 1
		}
		lines = append(lines, line)
	}

	if i > 0 {
		getDataByLines(p, lines)
	}
	
}

func getData(p Params, wg *sync.WaitGroup) {
	for _, f := range p.files {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()
			getDataByFile(p, file)
		}(f)
	}
}

func main(){
	var params Params 
	var wg sync.WaitGroup

	params.init()

	//ticker := time.NewTicker(time.Second * time.Duration(params.refresh))
	//exit := make(chan os.Signal, 1)
	//signal.Notify(exit, os.Interrupt, os.Kill)

	getData(params, &wg)

	wg.Wait()

}