package app

import (
	"fmt"

	"go.etcd.io/bbolt"
)

func connectBoltDB(path string) (*bbolt.DB, error) {
	var db, err = bbolt.Open(path, 0666, nil)
	if err != nil {
		fmt.Println(err, "connect to db failed")
		return nil, err
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		var _, err = tx.CreateBucketIfNotExists([]byte(dbBucketInfo))
		if err != nil {
			return fmt.Errorf("create bucket: %v", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}
