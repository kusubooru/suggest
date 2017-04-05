package boltstore

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"strconv"
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
			return fmt.Errorf("could not write NewAlias current bucket value to buffer: %v", werr)
		}

		if err := gob.NewDecoder(&buf).Decode(&alias); err != nil {
			// If err is anything else except io.EOF (empty value) we return.
			if err != io.EOF {
				return fmt.Errorf("could not decode current bucket value to alias: %v", err)
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
			return fmt.Errorf("could not encode alias: %v", err)
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
			return fmt.Errorf("could not write GetAliasOfUser current bucket value to buffer: %v", werr)
		}

		if err := gob.NewDecoder(&buf).Decode(&alias); err != nil {
			return fmt.Errorf("could not decode alias of user %v", err)
		}
		return nil
	})
	return alias, err
}

func (db *boltstore) DeleteAlias(username string, id uint64) error {
	var suggs []*teian.Alias
	buf := bytes.Buffer{}
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(aliasBucket))
		value := b.Get([]byte(username))

		if _, werr := buf.Write(value); werr != nil {
			return fmt.Errorf("could not write 'Delete value' to buffer: %v", werr)
		}

		if err := gob.NewDecoder(&buf).Decode(&suggs); err != nil {
			return fmt.Errorf("could not decode suggestions %v", err)
		}
		i, s := teian.SearchAliasByID(suggs, id)
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

func (db *boltstore) AllAlias() ([]*teian.Alias, error) {
	var allAlias []*teian.Alias
	buf := bytes.Buffer{}
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(aliasBucket))

		// Iterate over items in sorted key order.
		return b.ForEach(func(k, v []byte) error {
			var userAlias []*teian.Alias
			buf.Reset()
			if _, werr := buf.Write(v); werr != nil {
				return fmt.Errorf("could not write AllAlias current bucket value to buffer: %v", werr)
			}

			if err := gob.NewDecoder(&buf).Decode(&userAlias); err != nil {
				return fmt.Errorf("could not decode alias of %q: %v", k, err)
			}
			allAlias = append(allAlias, userAlias...)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return allAlias, nil
}
