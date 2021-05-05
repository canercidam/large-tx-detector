package badgerrepo

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/canercidam/large-tx-detector/core/agent"
	badger "github.com/dgraph-io/badger/v3"
)

// Config vars
var (
	DoneOperationTTL = time.Hour
)

// SaveOperation saves the operation.
func (repo *Repository) SaveOperation(op *agent.Operation) error {
	b, _ := json.Marshal(op)
	return repo.db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry(operationKey(op.TxHash, op.AgentID), b)
		if op.Done {
			entry = entry.WithTTL(DoneOperationTTL)
		}
		return txn.SetEntry(entry)
	})
}

// GetOperation gets the saved operation.
func (repo *Repository) GetOperation(txHash, agentID string) (*agent.Operation, error) {
	var op agent.Operation
	err := repo.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(operationKey(txHash, agentID))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &op)
		})
	})
	if err == badger.ErrKeyNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &op, nil
}

func operationKey(txHash, agentID string) []byte {
	return []byte(fmt.Sprintf("%s/%s", agentID, txHash))
}
