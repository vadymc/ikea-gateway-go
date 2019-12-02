package sql

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
	dbPath                   = "IKEA_DB_PATH"
	insertEventSQL           = "insert into event default values"
	insertStatDataSQL        = "insert into stat_data (event_id, group_name, power, dimmer, rgb, date_created) values (?, ?, ?, ?, ?, ?)"
	insertQuantileGroupSQL   = "insert into quantile_group (group_name, bucket_index, bucket_value) values (?, ?, ?)"
	selectQuantileGroupIDSQL = "select id, bucket_value from quantile_group where group_name=? and bucket_index=?"
	updateQuantileGroupIDSQL = "update quantile_group set bucket_value=? where id=?"
	selectRawEvents          = `SELECT distinct(e.id), e.date_created, group_name, dimmer
								FROM event e join stat_data sd on e.id = sd.event_id
								where e.date_created > ?
								order by e.id desc`
)

type IStorage interface {
	SaveGroupState(ctx context.Context, l []*LightState, wg *sync.WaitGroup)
	SaveQuantileGroup(g *QuantileGroup)
}

type DBStorage struct {
	db                        *sql.DB
	insertEventStmt           *sql.Stmt
	insertStatDataStmt        *sql.Stmt
	insertQuantileGroupStmt   *sql.Stmt
	selectQuantileGroupIDStmt *sql.Stmt
	updateQuantileGroupIDStmt *sql.Stmt
	selectRawDataStmt         *sql.Stmt
}

type LightState struct {
	Power    int
	Dimmer   int
	RGB      string
	Group    string
	Date     time.Time
	DeviceId int
}

type QuantileGroup struct {
	Name        string
	BucketIndex int
	BucketVal   int
}

type StatRow struct {
	Date      string
	GroupName string
	Dimmer    int
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

	stmt, err = db.Prepare(insertQuantileGroupSQL)
	if err != nil {
		log.WithError(err).Fatal("Failed to prepare insert quantile_group statement")
		return
	}
	s.insertQuantileGroupStmt = stmt

	stmt, err = db.Prepare(selectQuantileGroupIDSQL)
	if err != nil {
		log.WithError(err).Fatal("Failed to prepare select quantile_group statement")
		return
	}
	s.selectQuantileGroupIDStmt = stmt

	stmt, err = db.Prepare(updateQuantileGroupIDSQL)
	if err != nil {
		log.WithError(err).Fatal("Failed to prepare update quantile_group statement")
		return
	}
	s.updateQuantileGroupIDStmt = stmt

	stmt, err = db.Prepare(selectRawEvents)
	if err != nil {
		log.WithError(err).Fatal("Failed to prepare select raw events statement")
		return
	}
	s.selectRawDataStmt = stmt
}

func (s *DBStorage) SaveGroupState(ctx context.Context, lightGroup []*LightState, wg *sync.WaitGroup) {
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

func (s *DBStorage) SaveQuantileGroup(g *QuantileGroup) {
	var id int64
	var val int
	row := s.selectQuantileGroupIDStmt.QueryRow(g.Name, g.BucketIndex)
	row.Scan(&id, &val)
	if id != 0 && val == g.BucketVal { // value exists and did not change
		return
	} else if id != 0 { // value exists and has to be updated
		_, err := s.updateQuantileGroupIDStmt.Exec(g.BucketVal, id)
		if err != nil {
			log.WithError(err).WithField("QuantileGroup", &g).Error("Failed to update quantile_group")
		}
		log.WithFields(log.Fields{"g": g.Name, "i": g.BucketIndex, "old v": val, "new v": g.BucketVal}).Info("Updated QuantileGroup")
	} else { // value does not exist
		_, err := s.insertQuantileGroupStmt.Exec(g.Name, g.BucketIndex, g.BucketVal)
		if err != nil {
			log.WithError(err).WithField("QuantileGroup", &g).Error("Failed to insert quantile_group")
		}
		log.WithFields(log.Fields{"g": g.Name, "i": g.BucketIndex, "v": g.BucketVal}).Info("Inserted QuantileGroup")
	}
}

func (s *DBStorage) SelectRawData(startTime time.Time) *[]StatRow {
	rows, err := s.selectRawDataStmt.Query(startTime)
	if err != nil {
		log.WithError(err).Error("Failed to select raw data")
	}
	defer rows.Close()
	result := &[]StatRow{}
	for rows.Next() {
		var id int64
		var date string
		var name string
		var dimmer int
		err = rows.Scan(&id, &date, &name, &dimmer)
		if err != nil {
			log.WithError(err).Error("Failed to extract row")
			continue
		}
		*result = append(*result, StatRow{date, name, dimmer})
	}
	return result
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
