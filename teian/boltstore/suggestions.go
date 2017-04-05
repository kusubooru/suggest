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

func (db *boltstore) Create(username string, sugg *teian.Suggestion) error {

	err := db.Update(func(tx *bolt.Tx) error {
		var suggs []teian.Suggestion
		buf := bytes.Buffer{}

		// get bucket
		b := tx.Bucket([]byte(suggestionsBucket))

		// get current value and decode
		value := b.Get([]byte(username))

		if _, werr := buf.Write(value); werr != nil {
			return fmt.Errorf("could not write 'Create value' to buffer: %v", werr)
		}

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

func (db *boltstore) OfUser(username string) ([]teian.Suggestion, error) {
	var suggs []teian.Suggestion
	buf := bytes.Buffer{}
	err := db.View(func(tx *bolt.Tx) error {
		value := tx.Bucket([]byte(suggestionsBucket)).Get([]byte(username))

		if _, werr := buf.Write(value); werr != nil {
			return fmt.Errorf("could not write 'OfUser value' to buffer: %v", werr)
		}

		if err := gob.NewDecoder(&buf).Decode(&suggs); err != nil {
			return fmt.Errorf("could not decode suggestions %v", err)
		}
		return nil
	})
	return suggs, err
}

func (db *boltstore) Delete(username string, id uint64) error {
	var suggs []teian.Suggestion
	buf := bytes.Buffer{}
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(suggestionsBucket))
		value := b.Get([]byte(username))

		if _, werr := buf.Write(value); werr != nil {
			return fmt.Errorf("could not write 'Delete value' to buffer: %v", werr)
		}

		if err := gob.NewDecoder(&buf).Decode(&suggs); err != nil {
			return fmt.Errorf("could not decode suggestions %v", err)
		}
		i, s := teian.FindByID(suggs, id)
		if s == nil {
			return errors.New("entry does not exist")
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

func (db *boltstore) All() ([]teian.Suggestion, error) {
	var suggs []teian.Suggestion
	buf := bytes.Buffer{}
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(suggestionsBucket))

		// Iterate over items in sorted key order.
		return b.ForEach(func(k, v []byte) error {
			var userSuggs []teian.Suggestion
			buf.Reset()
			if _, werr := buf.Write(v); werr != nil {
				return fmt.Errorf("could not write 'All value' to buffer: %v", werr)
			}

			if err := gob.NewDecoder(&buf).Decode(&userSuggs); err != nil {
				return fmt.Errorf("could not decode suggestions of %q: %v", k, err)
			}
			suggs = append(suggs, userSuggs...)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return suggs, nil
}
