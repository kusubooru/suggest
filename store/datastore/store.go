package datastore

import (
	"database/sql"
	"log"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/kusubooru/teian/store"
)

type datastore struct {
	*sql.DB
}

// New creates a database connection for the given driver and configuration and
// returns a new Store.
func New(driver, config string) store.Store {
	db := Open(driver, config)
	return &datastore{db}
}

// Open opens a new database connection with the specified driver and
// connection string.
func Open(driver, config string) *sql.DB {
	db, err := sql.Open(driver, config)
	if err != nil {
		log.Println(err)
		log.Fatalln("database connection failed")
	}
	if driver == "mysql" {
		// per issue https://github.com/go-sql-driver/mysql/issues/257
		db.SetMaxIdleConns(0)
	}
	return db
}
