package badgerrepo

import (
	badger "github.com/dgraph-io/badger/v3"
)

// Repository interacts with the database.
type Repository struct {
	db *badger.DB
}

// New creates a new repository.
func New(path string) (*Repository, error) {
	var opt badger.Options
	if len(path) == 0 {
		opt = badger.DefaultOptions("").WithInMemory(true)
	} else {
		opt = badger.DefaultOptions(path)
	}
	db, err := badger.Open(opt)
	if err != nil {
		return nil, err
	}
	return &Repository{db}, nil
}

// Close implements io.Closer.
func (repo *Repository) Close() error {
	return repo.db.Close()
}
