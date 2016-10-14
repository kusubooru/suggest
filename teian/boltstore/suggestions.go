package boltstore

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/boltdb/bolt"
	"github.com/kusubooru/teian/teian"
)

func (db *datastore) Create(username string, sugg *teian.Sugg) error {

	err := db.boltdb.Update(func(tx *bolt.Tx) error {
		var suggs []teian.Sugg
		buf := bytes.Buffer{}

		// get bucket
		b := tx.Bucket([]byte(suggsBucket))

		// get current value and decode
		value := b.Get([]byte(username))
		buf.Write(value)
		if err := gob.NewDecoder(&buf).Decode(&suggs); err != nil {
			// If err is anything else except io.EOF (empty value) we return.
			if err != io.EOF {
				return fmt.Errorf("could not decode current bucket value: %v", err)
			}
		}

		// prepare new value
		id, err := b.NextSequence()
		if err != nil {
			return err
		}
		sugg.ID = id
		sugg.Username = username
		sugg.Created = time.Now()
		suggs = append(suggs, *sugg)

		// encode new value and store to bucket
		buf.Reset()
		if err := gob.NewEncoder(&buf).Encode(suggs); err != nil {
			return fmt.Errorf("could not encode new bucket value: %v", err)
		}
		_ = b.Put([]byte(username), buf.Bytes())
		return nil
	})
	return err
}

func (db *datastore) GetSugg(username string) ([]teian.Sugg, error) {
	var suggs []teian.Sugg
	buf := bytes.Buffer{}
	err := db.boltdb.View(func(tx *bolt.Tx) error {
		value := tx.Bucket([]byte(suggsBucket)).Get([]byte(username))
		buf.Write(value)
		if err := gob.NewDecoder(&buf).Decode(&suggs); err != nil {
			return fmt.Errorf("could not decode suggestions %v", err)
		}
		return nil
	})
	return suggs, err
}

func (db *datastore) Delete(username string, id uint64) error {
	var suggs []teian.Sugg
	buf := bytes.Buffer{}
	err := db.boltdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(suggsBucket))
		value := b.Get([]byte(username))
		buf.Write(value)
		if err := gob.NewDecoder(&buf).Decode(&suggs); err != nil {
			return fmt.Errorf("could not decode suggestions %v", err)
		}
		i, s := teian.FindByID(suggs, id)
		if s == nil {
			return errors.New("entry does not exit")
		}
		// delete from slice
		suggs = append(suggs[:i], suggs[i+1:]...)

		// encode remains and store to bucket
		buf.Reset()
		if err := gob.NewEncoder(&buf).Encode(suggs); err != nil {
			return fmt.Errorf("could not encode new value after delete: %v", err)
		}
		_ = b.Put([]byte(username), buf.Bytes())
		return nil
	})
	return err
}

func (db *datastore) All() ([]teian.Sugg, error) {
	var suggs []teian.Sugg
	buf := bytes.Buffer{}
	err := db.boltdb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(suggsBucket))

		// Iterate over items in sorted key order.
		if err := b.ForEach(func(k, v []byte) error {
			var userSuggs []teian.Sugg
			buf.Reset()
			buf.Write(v)
			if err := gob.NewDecoder(&buf).Decode(&userSuggs); err != nil {
				return fmt.Errorf("could not decode suggestions of %q: %v", k, err)
			}
			for _, sugg := range userSuggs {
				suggs = append(suggs, sugg)
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return suggs, nil
}
