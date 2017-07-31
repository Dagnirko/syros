package main

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/go-chi/jwtauth"
	"github.com/robfig/cron"
)

var version = "undefined"

func main() {
	var config = &Config{}
	flag.StringVar(&config.LogLevel, "LogLevel", "debug", "logging threshold level: debug|info|warn|error|fatal|panic")
	flag.IntVar(&config.Port, "Port", 8888, "HTTP port to listen on")
	flag.StringVar(&config.MongoDB, "MongoDB", "localhost:27017", "MongoDB server addresses comma delimited")
	flag.StringVar(&config.Database, "Database", "syros", "MongoDB database name")
	flag.StringVar(&config.JwtSecret, "JwtSecret", "syros", "JWT secret")
	flag.StringVar(&config.Credentials, "Credentials", "admin@admin", "Credentials format user@password")
	flag.StringVar(&config.AppPath, "AppPath", "", "Path to dist dir")
	flag.StringVar(&config.Nats, "Nats", "nats://localhost:4222", "Nats server addresses comma delimited")
	flag.Parse()

	setLogLevel(config.LogLevel)

	log.Infof("Starting with config: %+v", config)

	nc, err := NewNatsConnection(config.Nats, "syros-app")
	if err != nil {
		log.Fatalf("Nats connection error %v", err)
	}
	defer nc.Close()

	cronJob := cron.New()
	cronJob.Start()
	registry := NewRegistry(config, nc, cronJob)
	log.Infof("Register service as %v", registry.Agent.Id)
	registry.Register()

	if config.AppPath == "" {
		workDir, _ := os.Getwd()
		config.AppPath = filepath.Join(workDir, "dist")
		if config.LogLevel != "debug" {
			if _, err := os.Stat(filepath.Join(config.AppPath, "index.html")); err != nil {
				if os.IsNotExist(err) {
					log.Fatalf("index.html not found in %v", config.AppPath)
				} else {
					log.Fatalf("Path to dist dir %v error %v", config.AppPath, err.Error())
				}
			}
		}
	}

	repo, err := NewRepository(config)
	if err != nil {
		log.Fatalf("MongoDB connection error %v", err)
	}

	server := HttpServer{
		Config:     config,
		Repository: repo,
		TokenAuth:  jwtauth.New("HS256", []byte(config.JwtSecret), nil),
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
