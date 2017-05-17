package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/robfig/cron"
)

var version = "undefined"

func main() {
	var config = &Config{}
	flag.StringVar(&config.Environment, "Environment", "dev", "Environment dev|int|stg|test|prep|prod")
	flag.StringVar(&config.LogLevel, "LogLevel", "debug", "logging threshold level: debug|info|warn|error|fatal|panic")
	flag.IntVar(&config.Port, "Port", 8886, "HTTP port to listen on")
	flag.IntVar(&config.CollectInterval, "CollectInterval", 30, "Collect interval in seconds")
	flag.StringVar(&config.DockerApiAddresses, "DockerApiAddresses", "unix:///var/run/docker.sock", "Docker hosts API addresses comma delimited")
	flag.StringVar(&config.ConsulApiAddresses, "ConsulApiAddresses", "", "Consul hosts API addresses comma delimited")
	flag.StringVar(&config.VSphereApiAddress, "VSphereApiAddress", "", "VSphere API address")
	flag.StringVar(&config.VSphereInclude, "VSphereInclude", "", "VM include filter comma delimited")
	flag.StringVar(&config.VSphereExclude, "VSphereExclude", "", "VM exclude filter comma delimited")
	flag.IntVar(&config.VSphereCollectInterval, "VSphereCollectInterval", 55, "vSphere collect interval in seconds")
	flag.StringVar(&config.Nats, "Nats", "nats://localhost:4222", "Nats server addresses comma delimited")
	flag.Parse()

	setLogLevel(config.LogLevel)
	log.Infof("Starting with config: %+v", config)

	nc, err := NewNatsConnection(config.Nats, "syros-agent-"+config.Environment)
	if err != nil {
		log.Fatalf("Nats connection error %v", err)
	}
	defer nc.Close()

	cronJob := cron.New()

	registry := NewRegistry(config, nc, cronJob)
	log.Infof("Register service as %v", registry.Agent.Id)
	registry.Register()

	coordinator, err := NewCoordinator(config, nc, cronJob)
	if err != nil {
		log.Fatalf("Coordinator error %v", err)
	}
	coordinator.Register()
	defer coordinator.Deregister()

	server := &HttpServer{
		Config: config,
	}
	log.Infof("Starting HTTP server on port %v", config.Port)
	go server.Start()

	//wait for SIGINT (Ctrl+C) or SIGTERM (docker stop)
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	sig := <-sigChan
	log.Infof("Shutting down %v signal received", sig)
}

func setLogLevel(levelName string) {
	level, err := log.ParseLevel(levelName)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)
}
