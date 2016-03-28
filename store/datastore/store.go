package datastore

import (
	"database/sql"
	"log"
	"time"

	"github.com/boltdb/bolt"
	// mysql driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/kusubooru/teian/store"
)

const suggsBucket = "suggestions"

type datastore struct {
	*sql.DB
	boltdb *bolt.DB
}

// New creates a database connection for the given driver and configuration and
// returns a new Store.
func New(driver, config, boltFile string) store.Store {
	db := Open(driver, config)
	boltdb := OpenBolt(boltFile)
	return &datastore{db, boltdb}
}

func (db *datastore) Close() {
	if err := db.DB.Close(); err != nil {
		log.Print(err)
		log.Fatalln("database close failed")
	}
	if err := db.boltdb.Close(); err != nil {
		log.Print(err)
		log.Fatalln("bolt close failed")
	}
}

func OpenBolt(file string) *bolt.DB {
	db, err := bolt.Open(file, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		log.Print(err)
		log.Fatalln("bolt open failed")
	}
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(suggsBucket)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Print(err)
		log.Fatalln("bolt bucket creation failed")
	}
	return db
}

// Open opens a new database connection with the specified driver and
// connection string.
func Open(driver, config string) *sql.DB {
	db, err := sql.Open(driver, config)
	if err != nil {
		log.Print(err)
		log.Fatalln("database connection failed")
	}
	if driver == "mysql" {
		// per issue https://github.com/go-sql-driver/mysql/issues/257
		db.SetMaxIdleConns(0)
	}
	if err := pingDatabase(db); err != nil {
		log.Print(err)
		log.Fatalln("database ping attempts failed")
	}
	return db
}

// helper function to ping the database with backoff to ensure a connection can
// be established before we proceed.
func pingDatabase(db *sql.DB) (err error) {
	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			return
		}
		log.Print("database ping failed. retry in 1s")
		time.Sleep(time.Second)
	}
	return
}
