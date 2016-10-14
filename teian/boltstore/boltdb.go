package boltstore

import (
	"log"
	"time"

	"github.com/boltdb/bolt"
	"github.com/kusubooru/teian/teian"
)

const suggestionsBucket = "suggestions"

type boltstore struct {
	*bolt.DB
}

// Open opens the bolt database file and returns a new SuggestionStore. The
// bolt database file will be created if it does not exist.
func NewSuggestionStore(boltFile string) teian.SuggestionStore {
	boltdb := openBolt(boltFile)
	return &boltstore{boltdb}
}

func (db *boltstore) Close() {
	if err := db.DB.Close(); err != nil {
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
		if _, err = tx.CreateBucketIfNotExists([]byte(suggestionsBucket)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatalln("bolt bucket creation failed:", err)
	}
	return db
}
