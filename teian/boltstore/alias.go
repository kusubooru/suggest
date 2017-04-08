package boltstore

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/kusubooru/teian/teian"
)

func (db *boltstore) NewAlias(a *teian.Alias) error {
	err := db.Update(func(tx *bolt.Tx) error {
		// get bucket
		b := tx.Bucket([]byte(aliasBucket))

		// prepare new value
		id, err := b.NextSequence()
		if err != nil {
			return err
		}
		a.ID = id
		a.Created = time.Now()

		// encode new value and store to bucket
		buf := bytes.Buffer{}
		if err := gob.NewEncoder(&buf).Encode(a); err != nil {
			return fmt.Errorf("could not encode alias: %v", err)
		}
		_ = b.Put([]byte(strconv.FormatUint(id, 10)), buf.Bytes())
		return nil
	})
	return err
}

func (db *boltstore) UpdateAlias(id uint64, a *teian.Alias) (*teian.Alias, error) {
	alias, err := db.GetAlias(id)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		// get bucket
		b := tx.Bucket([]byte(aliasBucket))

		alias.Old = a.Old
		alias.New = a.New
		alias.Comment = a.Comment
		alias.Status = a.Status

		// encode new value and store to bucket
		buf := bytes.Buffer{}
		if err := gob.NewEncoder(&buf).Encode(alias); err != nil {
			return fmt.Errorf("could not encode alias: %v", err)
		}
		_ = b.Put([]byte(strconv.FormatUint(id, 10)), buf.Bytes())
		return nil
	})
	return alias, err
}

func (db *boltstore) GetAliasOfUser(username string) ([]*teian.Alias, error) {
	var userAlias []*teian.Alias
	buf := bytes.Buffer{}
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(aliasBucket))

		// Iterate over items in sorted key order.
		return b.ForEach(func(k, v []byte) error {
			var a *teian.Alias
			buf.Reset()
			if _, werr := buf.Write(v); werr != nil {
				return fmt.Errorf("could not write GetAliasOfUser current bucket value to buffer: %v", werr)
			}

			if err := gob.NewDecoder(&buf).Decode(&a); err != nil {
				return fmt.Errorf("could not decode alias of %q: %v", k, err)
			}
			if a.Username == username {
				userAlias = append(userAlias, a)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return userAlias, nil

}

func (db *boltstore) DeleteAlias(id uint64) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(aliasBucket))
		return b.Delete([]byte(strconv.FormatUint(id, 10)))
	})
}

func (db *boltstore) AllAlias() ([]*teian.Alias, error) {
	var allAlias []*teian.Alias
	buf := bytes.Buffer{}
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(aliasBucket))

		// Iterate over items in sorted key order.
		return b.ForEach(func(k, v []byte) error {
			var a *teian.Alias
			buf.Reset()
			if _, werr := buf.Write(v); werr != nil {
				return fmt.Errorf("could not write AllAlias current bucket value to buffer: %v", werr)
			}

			if err := gob.NewDecoder(&buf).Decode(&a); err != nil {
				return fmt.Errorf("could not decode alias of %q: %v", k, err)
			}
			allAlias = append(allAlias, a)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return allAlias, nil
}

func (db *boltstore) DeleteAllAlias() error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(aliasBucket))

		// Iterate over items in sorted key order.
		return b.ForEach(func(k, v []byte) error {
			return b.Delete(k)
		})
	})
}

// TODO: change alias to use id instead of username
func (db *boltstore) GetAlias(id uint64) (*teian.Alias, error) {
	var alias *teian.Alias
	buf := bytes.Buffer{}
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(aliasBucket))
		value := b.Get([]byte(strconv.FormatUint(id, 10)))
		if value == nil {
			return errors.New("alias does not exist")
		}

		if _, werr := buf.Write(value); werr != nil {
			return fmt.Errorf("could not write bucket data to buffer: %v", werr)
		}

		if err := gob.NewDecoder(&buf).Decode(&alias); err != nil {
			return fmt.Errorf("could not decode alias: %v", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return alias, nil
}
