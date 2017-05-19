package main

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"github.com/stefanprodan/syros/models"
)

type DockerCollector struct {
	ApiAddress  string
	Environment string
	Topic       string
	StopChan    chan bool
}

func NewDockerCollector(address string, env string) (*DockerCollector, error) {

	collector := &DockerCollector{
		ApiAddress:  address,
		Environment: env,
		Topic:       "docker",
		StopChan:    make(chan bool, 1),
	}

	return collector, nil
}

func (col *DockerCollector) Collect() (*models.DockerPayload, error) {
	start := time.Now().UTC()
	payload := &models.DockerPayload{}
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	client, err := docker.NewClient(col.ApiAddress, "", nil, defaultHeaders)
	defer client.Close()
	if err != nil {
		return nil, err
	}
	host, err := client.Info(context.Background())
	if err != nil {
		return nil, err
	}
	payload.Host = MapDockerHost(col.Environment, host)

	options := types.ContainerListOptions{All: true}
	containers, err := client.ContainerList(context.Background(), options)
	if err != nil {
		return nil, err
	}

	payload.Containers = make([]models.DockerContainer, 0)

	for _, container := range containers {
		containerInfo, err := client.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			log.Error(err)
			continue
		}
		payload.Containers = append(payload.Containers, MapDockerContainer(col.Environment, host.ID, host.Name, container, containerInfo))
	}

	log.Debugf("%v collect duration: %v containers %v", col.ApiAddress, time.Now().UTC().Sub(start), len(payload.Containers))
	return payload, nil
}

func MapDockerHost(environment string, info types.Info) models.DockerHost {
	host := models.DockerHost{
		Id:                 models.Hash(info.Name),
		Containers:         info.Containers,
		ContainersRunning:  info.ContainersRunning,
		ContainersPaused:   info.ContainersPaused,
		ContainersStopped:  info.ContainersStopped,
		Images:             info.Images,
		Driver:             info.Driver,
		SystemTime:         info.SystemTime,
		LoggingDriver:      info.LoggingDriver,
		CgroupDriver:       info.CgroupDriver,
		NEventsListener:    info.NEventsListener,
		KernelVersion:      info.KernelVersion,
		OperatingSystem:    info.OperatingSystem,
		OSType:             info.OSType,
		Architecture:       info.Architecture,
		IndexServerAddress: info.IndexServerAddress,
		NCPU:               info.NCPU,
		MemTotal:           info.MemTotal,
		DockerRootDir:      info.DockerRootDir,
		HTTPProxy:          info.HTTPProxy,
		HTTPSProxy:         info.HTTPSProxy,
		NoProxy:            info.NoProxy,
		Name:               info.Name,
		Labels:             info.Labels,
		ExperimentalBuild:  info.ExperimentalBuild,
		ServerVersion:      info.ServerVersion,
		ClusterStore:       info.ClusterStore,
		ClusterAdvertise:   info.ClusterAdvertise,
		DefaultRuntime:     info.DefaultRuntime,
		LiveRestoreEnabled: info.LiveRestoreEnabled,
		Collected:          time.Now().UTC(),
		Environment:        environment,
	}

	for _, reg := range info.RegistryConfig.IndexConfigs {
		host.Registries = append(host.Registries, reg.Name)
	}

	return host
}

func MapDockerContainer(environment string, hostId string, hostName string, c types.Container, cj types.ContainerJSON) models.DockerContainer {
	container := models.DockerContainer{
		Id:           c.ID,
		HostId:       models.Hash(hostName),
		HostName:     hostName,
		Image:        c.Image,
		Command:      c.Command,
		State:        c.State,
		Status:       c.Status,
		Path:         cj.ContainerJSONBase.Path,
		Args:         cj.ContainerJSONBase.Args,
		Name:         cj.ContainerJSONBase.Name,
		RestartCount: cj.ContainerJSONBase.RestartCount,
		PortBindings: make(map[string]string),
		Collected:    time.Now().UTC(),
		Environment:  environment,
	}

	container.Labels = make(map[string]string)
	for key, value := range c.Labels {
		k := strings.Replace(key, ".", "_", -1)
		container.Labels[k] = value
	}

	container.Created, _ = time.Parse(time.RFC3339, cj.ContainerJSONBase.Created)
	if len(container.Name) > 1 {
		container.Name = container.Name[1:len(container.Name)]
	}

	if cj.Config != nil {
		container.Env = cj.Config.Env
	}

	if cj.ContainerJSONBase.State != nil {

		container.StartedAt, _ = time.Parse(time.RFC3339, cj.ContainerJSONBase.State.StartedAt)
		container.FinishedAt, _ = time.Parse(time.RFC3339, cj.ContainerJSONBase.State.FinishedAt)
		container.ExitCode = cj.ContainerJSONBase.State.ExitCode
		container.Error = cj.ContainerJSONBase.State.Error
	}

	if cj.ContainerJSONBase.HostConfig != nil {
		container.NetworkMode = string(cj.ContainerJSONBase.HostConfig.NetworkMode)
		container.RestartPolicy = cj.ContainerJSONBase.HostConfig.RestartPolicy.Name
		for key, val := range cj.ContainerJSONBase.HostConfig.PortBindings {
			if len(val) > 0 {
				container.PortBindings[string(key)] = val[0].HostPort
			}
		}
	}

	// use first Host port bind as container Port
	// if multiple ports are present search for gliderlabs/registrator meta
	container.Port = GetPortFromEnv(container.PortBindings, container.Env)

	return container
}

func GetPortFromEnv(portBindings map[string]string, env []string) string {
	port := ""
	if len(portBindings) == 0 {
		return port
	} else if len(portBindings) == 1 {
		for _, val := range portBindings {
			if len(val) > 0 {
				return val
			}
		}
	}
	if len(env) == 0 {
		for _, val := range portBindings {
			if len(val) > 0 {
				return val
			}
		}
	}
	for _, val := range portBindings {
		if len(val) > 0 {
			registratorMeta := "SERVICE_" + val + "_NAME"
			for _, e := range env {
				if strings.Contains(e, registratorMeta) {
					return val
				}
			}
		}
	}
	if port == "" {
		for _, val := range portBindings {
			if len(val) > 0 {
				return val
			}
		}
	}

	return port
}
