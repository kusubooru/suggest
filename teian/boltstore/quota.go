package boltstore

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/boltdb/bolt"
	"github.com/kusubooru/teian/teian"
)

func (db *Boltstore) CheckQuota(username string, n teian.Quota) (teian.Quota, error) {

	var remain teian.Quota

	err := db.Update(func(tx *bolt.Tx) error {
		var usage teian.Quota
		buf := bytes.Buffer{}

		// get bucket
		b := tx.Bucket([]byte(quotaBucket))

		// get current value and decode
		value := b.Get([]byte(username))

		if _, werr := buf.Write(value); werr != nil {
			return fmt.Errorf("could not write 'Create value' to buffer: %v", werr)
		}

		if err := gob.NewDecoder(&buf).Decode(&usage); err != nil {
			// If err is anything else except io.EOF (empty value) we return.
			if err != io.EOF {
				return fmt.Errorf("could not decode current bucket value: %v", err)
			}
		}

		max := int64(db.userQuota)
		newUsageInt64 := int64(usage) + int64(n)

		if newUsageInt64 > max {
			return teian.ErrOverQuota
		}
		newUsage := teian.Quota(newUsageInt64)
		remain = teian.Quota(max - newUsageInt64)

		// encode new value and store to bucket
		buf.Reset()
		if err := gob.NewEncoder(&buf).Encode(newUsage); err != nil {
			return fmt.Errorf("could not encode new bucket value: %v", err)
		}
		_ = b.Put([]byte(username), buf.Bytes())
		return nil
	})
	return remain, err
}

func (db *Boltstore) CleanQuota() error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(quotaBucket))
		return b.ForEach(func(k, v []byte) error {
			if err := b.Delete(k); err != nil {
				return err
			}
			return nil
		})
	})
}
