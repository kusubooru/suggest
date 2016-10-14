package boltstore

import (
	"log"
	"time"

	"github.com/boltdb/bolt"
	"github.com/kusubooru/teian/teian"
)

const suggsBucket = "suggestions"

type datastore struct {
	boltdb *bolt.DB
}

// Open opens the bolt database file and returns a new SuggStore. The bolt
// database file will be created if it does not exist.
func Open(boltFile string) teian.SuggestionStore {
	boltdb := openBolt(boltFile)
	return &datastore{boltdb}
}

func (db *datastore) Close() {
	if err := db.boltdb.Close(); err != nil {
		log.Fatalln("bolt close failed:", err)
	}
}

// openBolt creates and opens a bolt database at the given path. If the file does
// not exist then it will be created automatically. After opening it creates
// all the needed buckets.
func openBolt(file string) *bolt.DB {
	db, err := bolt.Open(file, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		log.Fatalln("bolt open failed:", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err = tx.CreateBucketIfNotExists([]byte(suggsBucket)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatalln("bolt bucket creation failed:", err)
	}
	return db
}
