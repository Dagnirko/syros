package models

import "time"

type Release struct {
	Id           string    `bson:"_id,omitempty" json:"id"`
	TicketId     string    `bson:"ticket_id,omitempty" json:"ticket_id"`
	Name         string    `bson:"name" json:"name"`
	Begin        time.Time `bson:"begin" json:"begin"`
	End          time.Time `bson:"end" json:"end"`
	Deployments  int       `bson:"deployments" json:"deployments"`
	Environments string    `bson:"environments" json:"environments"`
	Log          string    `bson:"log" json:"log"`
}

type Deployment struct {
	Id          string            `bson:"_id,omitempty" json:"id"`
	ReleaseId   string            `bson:"release_id,omitempty" json:"release_id"`
	TicketId    string            `bson:"ticket_id,omitempty" json:"ticket_id"`
	Status      string            `bson:"status" json:"status"`
	ServiceName string            `bson:"service_name" json:"service_name"`
	HostName    string            `bson:"host_name" json:"host_name"`
	Environment string            `bson:"environment" json:"environment"`
	Timestamp   time.Time         `bson:"timestamp" json:"timestamp"`
	Image       string            `bson:"image" json:"image"`
	Command     string            `bson:"command" json:"command"`
	Labels      map[string]string `bson:"labels" json:"labels"`
	Env         []string          `bson:"env" json:"env"`
	Log         string            `bson:"log" json:"log"`
}

type ReleasePlan struct {
	Id        string    `bson:"_id,omitempty" json:"id"`
	TicketId  string    `bson:"ticket_id,omitempty" json:"ticket_id"`
	Name      string    `bson:"name" json:"name"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
}

type ReleaseStep struct {
	Id          string    `bson:"_id,omitempty" json:"id"`
	ReleaseId   string    `bson:"releaseplan_id,omitempty" json:"releaseplan_id"`
	TicketId    string    `bson:"ticket_id,omitempty" json:"ticket_id"`
	Order       int       `bson:"order" json:"order"`
	Environment string    `bson:"environment" json:"environment"`
	Command     string    `bson:"command" json:"command"`
	Status      string    `bson:"status" json:"status"`
	Log         string    `bson:"log" json:"log"`
	Begin       time.Time `bson:"begin" json:"begin"`
	End         time.Time `bson:"end" json:"end"`
}
