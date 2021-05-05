package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/canercidam/large-tx-detector/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// Errors
var (
	ErrIgnore = errors.New("agent ignores the transaction")
)

// Agent is a transaction handler with iteration capabilities.
type Agent interface {
	ID() string
	Skip(*types.Block, *types.Transaction) bool
	Init(*Operation, *types.Transaction)
	Next() bool
	core.TransactionHandler
}

// AgentRepository manages agent operations i.e. tx handling per agent.
type AgentRepository interface {
	SaveOperation(*Operation) error
	GetOperation(txHash, agentID string) (*Operation, error)
}

// Pool aggregates registered agents and handles a transaction for each.
type Pool struct {
	agents []Agent
	repo   AgentRepository
}

// NewPool creates a new pool.
func NewPool(repo AgentRepository) *Pool {
	return &Pool{repo: repo}
}

// AddAgent registers and agent to handle any incoming tx.
func (pool *Pool) AddAgent(agent Agent) {
	pool.agents = append(pool.agents, agent)
}

// HandleTransaction implements core.TransactionHandler.
func (pool *Pool) HandleTransaction(ctx context.Context, block *types.Block, tx *types.Transaction) error {
	for _, agent := range pool.agents {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := pool.handleTxWithAgent(ctx, block, tx, agent); err != nil {
				return fmt.Errorf("agent '%s' failed: %v", agent.ID(), err)
			}
		}
	}
	return nil
}

func (pool *Pool) handleTxWithAgent(ctx context.Context, block *types.Block, tx *types.Transaction, agent Agent) error {
	if agent.Skip(block, tx) {
		return nil
	}

	op, err := pool.repo.GetOperation(tx.Hash().String(), agent.ID())
	if err != nil {
		return err
	}
	if op != nil && op.Done {
		return nil
	}
	if op == nil {
		op = &Operation{
			TxHash:      tx.Hash().String(),
			BlockNumber: block.NumberU64(),
			AgentID:     agent.ID(),
		}
	}

	// Iterate over the agent actions until the sequence has been completed.
	agent.Init(op, tx)
	var txErr error
	for {
		if !agent.Next() {
			op.Done = true
			break
		}
		txErr = agent.HandleTransaction(ctx, block, tx)
		if err != nil {
			break
		}
		op.State++
	}

	// No need to save the ignored operations.
	if txErr == ErrIgnore {
		return nil
	}

	if err := pool.repo.SaveOperation(op); err != nil {
		return fmt.Errorf("failed to save the operation: %v", err)
	}

	return txErr
}
