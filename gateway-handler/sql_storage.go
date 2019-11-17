package gateway_handler

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

const (
	dbPath            = "IKEA_DB_PATH"
	insertEventSQL    = "insert into event default values"
	insertStatDataSQL = "insert into stat_data (event_id, group_name, power, dimmer, rgb, date_created) values (?, ?, ?, ?, ?, ?)"
)

type IStorage interface {
	SaveGroupState(ctx context.Context, l []LightState, wg *sync.WaitGroup)
}

type DBStorage struct {
	db                 *sql.DB
	insertEventStmt    *sql.Stmt
	insertStatDataStmt *sql.Stmt
}

func NewDBStorage() *DBStorage {
	s := DBStorage{}
	s.init()
	return &s
}

func (s *DBStorage) init() {
	db, err := sql.Open("sqlite3", os.Getenv(dbPath))
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to DB")
		return
	}
	s.db = db

	stmt, err := db.Prepare(insertEventSQL)
	if err != nil {
		log.WithError(err).Fatal("Failed to prepare insert event statement")
		return
	}
	s.insertEventStmt = stmt

	stmt, err = db.Prepare(insertStatDataSQL)
	if err != nil {
		log.WithError(err).Fatal("Failed to prepare insert stat_data statement")
		return
	}
	s.insertStatDataStmt = stmt
}

func (s *DBStorage) SaveGroupState(ctx context.Context, lightGroup []LightState, wg *sync.WaitGroup) {
	start := time.Now()
	defer func() {
		wg.Done()
		log.WithField("SaveGroupState", "sql storage").WithField("elapsed time", time.Since(start)).Info("Done")
	}()
	err := s.withTransaction(func() error {
		r, err := s.insertEventStmt.ExecContext(ctx)
		if err != nil {
			log.WithError(err).WithField("lightGroup", lightGroup).Error("Failed to insert r")
			return err
		}
		eventId, _ := r.LastInsertId()
		for _, ls := range lightGroup {
			_, err := s.insertStatDataStmt.ExecContext(ctx, eventId, ls.Group, ls.Power, ls.Dimmer, ls.RGB, ls.Date)
			if err != nil {
				log.WithError(err).WithField("LightState", ls).Error("Failed to insert stat_data")
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.WithError(err).WithField("lightGroup", lightGroup).Error("Failed to SaveGroupState")
	}
}

func (s *DBStorage) withTransaction(f func() error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	err = f()
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
