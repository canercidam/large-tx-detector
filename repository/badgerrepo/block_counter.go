package badgerrepo

import (
	"strconv"

	badger "github.com/dgraph-io/badger/v3"
)

const (
	latestBlockKey = "latest-block"
)

// GetLatestBlock gets the latest block from the database.
func (repo *Repository) GetLatestBlock() (uint64, error) {
	var latestBlock uint64
	err := repo.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(latestBlockKey))
		if err != nil {
			return nil
		}
		return item.Value(func(val []byte) error {
			// We ignore the error because we will consider zero as acceptable.
			latestBlock, _ = strconv.ParseUint(string(val), 10, 64)
			return nil
		})
	})
	if err == badger.ErrKeyNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return latestBlock, nil
}

// SetLatestBlock sets the latest block in the database.
func (repo *Repository) SetLatestBlock(latestBlock uint64) error {
	return repo.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(latestBlockKey), []byte(strconv.FormatUint(latestBlock, 10)))
	})
}
