package main

import (
	"log"
	"strings"

	"path"

	"github.com/urfave/cli"
)

func componentPromote(c *cli.Context) error {

	config := c.GlobalString("config")
	ticket := c.String("ticket")
	environments := c.StringSlice("environment")
	components := c.StringSlice("component")
	tag := c.String("tag")

	dir, err := createArtifactsDir("/tmp")
	if err != nil {
		panic(err.Error())
	}

	setLogFile(dir)

	log.Print(">>> Deployment started")
	err = downloadArtifacts(config, dir)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("Config downloaded to %s", dir)
	log.Print("-----------------")

	for _, component := range components {
		for _, env := range environments {

			componentCfg, exists, err := loadComponent(dir, env, component)
			if err != nil {
				log.Fatal(err.Error())
			}
			if !exists {
				log.Printf("No config found for %s on %s", component, env)
				log.Print("-----------------")
				continue
			}

			promotionCfg, err := loadPromotion(dir, env)
			if err != nil {
				log.Fatal(err.Error())
			}

			if componentCfg.Component.Type == "docker" {
				for _, target := range componentCfg.Component.Target {

					if len(ticket) > 0 {
						syrosApi, cfgExists, err := loadSyrosConfig(dir, "syros")
						if err != nil {
							log.Printf("Syros config load failed %s", err.Error())
						} else {
							if !cfgExists {
								log.Print("Syros config not found")
							} else {
								err := syrosApi.Start(ticket, env, component, target.Host)
								if err != nil {
									log.Print(err.Error())
								}
							}
						}
					}

					fromEnv := promotionCfg.Rules.Source
					hostFrom := strings.Replace(target.Host, env, fromEnv, 1)
					cd := ContainerDeploy{
						Dir:      dir,
						Env:      env,
						HostFrom: hostFrom,
						HostTo:   target.Host,
						Service:  component,
						Tag:      tag,
						Ticket:   ticket,
						Check:    target.Health,
					}

					err = cd.Promote()
					if err != nil {
						log.Fatal(err.Error())
					}

					if len(ticket) > 0 {
						jira, cfgExists, err := loadJiraConfig(dir, "jira")
						if err != nil {
							log.Printf("Jira config load failed %s", err.Error())
						} else {
							if !cfgExists {
								log.Print("Jira config not found")
							} else {
								err := jira.Post(ticket, env, component, target.Host)
								if err != nil {
									log.Print(err.Error())
								}
							}
						}
						syrosApi, cfgExists, err := loadSyrosConfig(dir, "syros")
						if err != nil {
							log.Printf("Syros config load failed %s", err.Error())
						} else {
							if !cfgExists {
								log.Print("Syros config not found")
							} else {
								err := syrosApi.Finish(ticket, env, component, target.Host, path.Join(dir, "deployctl.log"))
								if err != nil {
									log.Print(err.Error())
								}
							}
						}
					}

					log.Printf("Deployment complete for %s on %s", component, target.Host)
					log.Print("-----------------")
				}
			}
		}
	}

	if len(ticket) > 0 {
		jira, cfgExists, err := loadJiraConfig(dir, "jira")
		if err != nil {
			log.Printf("Jira config load failed %s", err.Error())
		} else {
			if !cfgExists {
				log.Print("Jira config not found")
			} else {
				err := jira.Upload(ticket, dir, "deployctl.log")
				if err != nil {
					log.Print(err.Error())
				}
			}
		}
	}

	log.Print(">>> Deployment complete")

	return nil
}
