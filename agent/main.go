package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var config = &Config{}
	flag.StringVar(&config.Environment, "Environment", "dev", "Environment dev|int|stg|test|prep|prod")
	flag.StringVar(&config.LogLevel, "LogLevel", "debug", "logging threshold level: debug|info|warn|error|fatal|panic")
	flag.IntVar(&config.Port, "Port", 8886, "HTTP port to listen on")
	flag.IntVar(&config.CollectInterval, "CollectInterval", 10, "Collect interval in seconds")
	flag.StringVar(&config.DockerApiAddresses, "DockerApiAddresses", "unix:///var/run/docker.sock", "Docker hosts API addresses comma delimited")
	flag.StringVar(&config.ConsulApiAddresses, "ConsulApiAddresses", "", "Consul hosts API addresses comma delimited")
	flag.StringVar(&config.Nats, "Nats", "nats://localhost:4222", "Nats server addresses comma delimited")
	flag.Parse()

	setLogLevel(config.LogLevel)
	log.Infof("Starting with config: %+v", config)

	nc, err := NewNatsConnection(config.Nats)
	defer nc.Close()
	if err != nil {
		log.Fatalf("Nats connection error %v", err)
	}
	log.Infof("Connected to NATS server %v status %v", nc.ConnectedUrl(), nc.Status())

	registry := NewRegistry(config, nc)
	log.Infof("Register service as %v", registry.Agent.Id)
	go func(r *Registry) {
		for true {
			err := r.RegisterAgent()
			if err != nil {
				log.Error(err)
			}
			time.Sleep(10 * time.Second)
		}
	}(registry)

	coordinator, err := NewCoordinator(config, nc)
	if err != nil {
		log.Fatalf("Coordinator error %v", err)
	}
	coordinator.StartCollectors()

	server := &HttpServer{
		Config: config,
		Status: coordinator.Status,
	}
	log.Infof("Starting HTTP server on port %v", config.Port)
	go server.Start()

	//wait for SIGINT (Ctrl+C) or SIGTERM (docker stop)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	log.Infof("Shuting down %v signal received", sig)
	server.Stop()
	coordinator.StopCollectors()
	time.Sleep(10 * time.Second)
}

func setLogLevel(levelName string) {
	level, err := log.ParseLevel(levelName)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)
}
