package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/stefanprodan/syros/models"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"strings"
	"time"
)

type MongoRepository struct {
	Config  *Config
	Session *mgo.Session
}

func NewMongoRepository(config *Config) (*MongoRepository, error) {
	cluster := strings.Split(config.MongoDB, ",")
	dialInfo := &mgo.DialInfo{
		Addrs:    cluster,
		Timeout:  60 * time.Second,
		Database: config.Database,
	}

	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		return nil, err
	}

	session.SetMode(mgo.Monotonic, true)

	repo := &MongoRepository{
		Config:  config,
		Session: session,
	}

	return repo, nil
}

func (repo *MongoRepository) Initialize() {
	repo.CreateIndex("hosts", "environment")
	repo.CreateIndex("hosts", "collected")
	repo.CreateIndex("containers", "host_id")
	repo.CreateIndex("containers", "environment")
	repo.CreateIndex("containers", "collected")
	repo.CreateIndex("checks", "host_id")
	repo.CreateIndex("checks", "environment")
	repo.CreateIndex("checks", "collected")
	repo.CreateIndex("syros_services", "environment")
	repo.CreateIndex("syros_services", "collected")
}

func (repo *MongoRepository) CreateIndex(col string, index string) {
	c := repo.Session.DB(repo.Config.Database).C(col)
	err := c.EnsureIndexKey(index)

	if err != nil {
		log.Fatalf("MongoDB index %v init failed %v", index, err)
	}
}

func (repo *MongoRepository) SyrosServiceUpsert(service models.SyrosService) {
	s := repo.Session.Copy()
	defer s.Close()

	c := s.DB(repo.Config.Database).C("syros_services")

	_, err := c.UpsertId(service.Id, &service)
	if err != nil {
		log.Errorf("Repository syros_services upsert failed %v", err)
	}
}

// Removes stale records
func (repo *MongoRepository) RunGarbageCollector(cols []string) {
	if repo.Config.DatabaseStale > 0 {
		log.Infof("Stating repository GC interval %v minutes", repo.Config.DatabaseStale)
		go func(stale int) {

			for true {
				s := repo.Session.Copy()
				for _, col := range cols {
					c := s.DB(repo.Config.Database).C(col)
					info, err := c.RemoveAll(
						bson.M{
							"collected": bson.M{
								"$lt": time.Now().Add(-time.Duration(stale) * time.Minute).UTC(),
							},
						})
					if err != nil {
						log.Errorf("Repository GC for col %v query failed %v", col, err)
					} else {
						if info.Removed > 0 {
							log.Infof("Repository GC removed %v from %v", info.Removed, col)
						}
					}
				}
				s.Close()
				time.Sleep(60 * time.Second)
			}

		}(repo.Config.DatabaseStale)
	}
}
