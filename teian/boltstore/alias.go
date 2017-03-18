package boltstore

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"time"

	"github.com/boltdb/bolt"
	"github.com/kusubooru/teian/teian"
)

func (db *boltstore) NewAlias(username string, a *teian.Alias) error {

	err := db.Update(func(tx *bolt.Tx) error {
		var alias []teian.Alias
		buf := bytes.Buffer{}

		// get bucket
		b := tx.Bucket([]byte(aliasBucket))

		// get current value and decode
		value := b.Get([]byte(username))

		if _, werr := buf.Write(value); werr != nil {
			return fmt.Errorf("alias: could not write 'NewAlias value' to buffer: %v", werr)
		}

		if err := gob.NewDecoder(&buf).Decode(&alias); err != nil {
			// If err is anything else except io.EOF (empty value) we return.
			if err != io.EOF {
				return fmt.Errorf("alias: could not decode current bucket value: %v", err)
			}
		}

		// prepare new value
		id, err := b.NextSequence()
		if err != nil {
			return err
		}
		a.ID = id
		a.Username = username
		a.Created = time.Now()
		alias = append(alias, *a)

		// encode new value and store to bucket
		buf.Reset()
		if err := gob.NewEncoder(&buf).Encode(alias); err != nil {
			return fmt.Errorf("alias: could not encode new bucket value: %v", err)
		}
		_ = b.Put([]byte(username), buf.Bytes())
		return nil
	})
	return err
}

func (db *boltstore) GetAliasOfUser(username string) ([]*teian.Alias, error) {
	var alias []*teian.Alias
	buf := bytes.Buffer{}
	err := db.View(func(tx *bolt.Tx) error {
		value := tx.Bucket([]byte(aliasBucket)).Get([]byte(username))

		if _, werr := buf.Write(value); werr != nil {
			return fmt.Errorf("alias: could not write 'GetAliasOfUser value' to buffer: %v", werr)
		}

		if err := gob.NewDecoder(&buf).Decode(&alias); err != nil {
			return fmt.Errorf("alias: could not decode alias %v", err)
		}
		return nil
	})
	return alias, err
}
