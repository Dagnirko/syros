package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"strings"
	"time"
)

func main() {
	var config = &Config{}
	flag.StringVar(&config.LogLevel, "LogLevel", "debug", "logging threshold level: debug|info|warn|error|fatal|panic")
	flag.IntVar(&config.Port, "Port", 8000, "HTTP port to listen on")
	flag.IntVar(&config.CollectInterval, "CollectInterval", 10, "Collect interval in seconds")
	flag.StringVar(&config.Hosts, "Hosts", "", "Docker hosts API addresses comma delimited")
	flag.Parse()

	setLogLevel(config.LogLevel)
	log.Infof("Config: %+v", config)

	hosts := strings.Split(config.Hosts, ",")
	if len(hosts) < 1 {
		log.Fatalf("no hosts supplied %s", config.Hosts)
	}

	collectors := make([]*DockerCollector, len(hosts))
	for i, host := range hosts {
		collector, err := NewDockerCollector(host)
		if err != nil {
			log.Fatal(err)
		}
		collectors[i] = collector
	}

	log.Infof("Starting %v collector(s), collect interval is set to %v second(s)", len(collectors), config.CollectInterval)
	for _, c := range collectors {
		go func(collector *DockerCollector) {
			stop := false
			for !stop {
				select {
				case <-collector.StopChan:
					stop = true
				default:
					payload, err := collector.Collect()
					if err != nil {
						log.Error(err)
					} else {
						fmt.Printf("Host %+v containers running %+v\n", payload.Host.Name, payload.Host.ContainersRunning)
					}
					time.Sleep(time.Duration(config.CollectInterval) * time.Second)
				}
			}
			log.Infof("Collector exited %v", collector.Host)
		}(c)
	}

	//wait for SIGINT (Ctrl+C) or SIGTERM (docker stop)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	log.Infof("Shuting down %v signal received", sig)
	log.Infof("Stopping %v collector(s)", len(collectors))
	for _, collector := range collectors {
		collector.StopChan <- struct{}{}
	}
	time.Sleep(10 * time.Second)
}

func setLogLevel(levelName string) {
	level, err := log.ParseLevel(levelName)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)
}
