package main

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

type Repository struct {
	Config  *Config
	Session *r.Session
}

func NewRepository(config *Config) (*Repository, error) {

	session, err := r.Connect(r.ConnectOpts{
		Address:  config.RethinkDB,
		Database: config.Database,
	})
	if err != nil {
		return nil, err
	}

	repo := &Repository{
		Config:  config,
		Session: session,
	}
	return repo, nil
}

func (repo *Repository) Initialize() {
	var cursor *r.Cursor
	var err error
	var cnt int

	// init db
	cursor, err = r.DBList().Contains(repo.Config.Database).Run(repo.Session)
	if err != nil {
		log.Fatalf("RethinkDB database init query failed %v", err)
	}

	cursor.One(&cnt)
	cursor.Close()

	if cnt < 1 {
		log.Infof("No database found, creating %v", repo.Config.Database)
		_, err := r.DBCreate(repo.Config.Database).RunWrite(repo.Session)
		if err != nil {
			log.Fatalf("RethinkDB database creation failed %v", err)
		}
	}

	cursor, err = r.DB(repo.Config.Database).TableList().Contains("hosts").Run(repo.Session)
	if err != nil {
		log.Fatalf("RethinkDB table init query failed %v", err)
	}

	cursor.One(&cnt)
	cursor.Close()

	if cnt < 1 {
		log.Infof("No table found, creating %v", "hosts")
		_, err := r.DB(repo.Config.Database).TableCreate("hosts").RunWrite(repo.Session)
		if err != nil {
			log.Fatalf("RethinkDB &v table creation failed %v", "hosts", err)
		}
	}
}

func (repo *Repository) HostUpsert(host DockerHost) {
	res, err := r.Table("hosts").Get(host.Id).Run(repo.Session)
	if err != nil {
		log.Errorf("Repository host upsert query after ID failed %v", err)
	}

	if res.IsNil() {
		_, err := r.Table("hosts").Insert(host).RunWrite(repo.Session)
		if err != nil {
			log.Errorf("Repository host insert failed %v", err)
		}
	} else {
		_, err := r.Table("hosts").Get(host.Id).Update(host).Run(repo.Session)
		if err != nil {
			log.Errorf("Repository host update failed %v", err)
		}
	}
}

func logRepositoryResponse(action string, response interface{}) {
	jBytes, _ := json.Marshal(response)
	log.Debugf("Repository %v result %s", action, string(jBytes))
}
