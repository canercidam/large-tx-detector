package core

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
)

// Config vars
var (
	BlockConsumerBackOff = time.Second * 5
)

// BlockchainListener listens to the blockchain and provides the new blocks.
type BlockchainListener interface {
	ListenToNewBlocks(ctx context.Context, startBlock ...uint64) (<-chan *types.Block, error)
	io.Closer
}

// TransactionHandler handles a transaction.
type TransactionHandler interface {
	HandleTransaction(context.Context, *types.Block, *types.Transaction) error
}

// BlockCounter keeps track of the blocks we need to process.
type BlockCounter interface {
	GetLatestBlock() (uint64, error)
	SetLatestBlock(uint64) error
}

// BlockConsumer listens to the blockchain and consumes the new transactions
// in the new blocks.
type BlockConsumer struct {
	bcListener   BlockchainListener
	txHandler    TransactionHandler
	blockCounter BlockCounter

	ch <-chan *types.Block
}

// NewBlockConsumer creates a new block consumer.
func NewBlockConsumer(bcListener BlockchainListener, txHandler TransactionHandler, blockCounter BlockCounter) *BlockConsumer {
	return &BlockConsumer{bcListener: bcListener, txHandler: txHandler, blockCounter: blockCounter}
}

// Start starts the block consumer.
func (blCons *BlockConsumer) Start(ctx context.Context) (err error) {
	latestBlock, err := blCons.blockCounter.GetLatestBlock()
	if err != nil {
		return err
	}
	blCons.ch, err = blCons.bcListener.ListenToNewBlocks(ctx, latestBlock)
	if err != nil {
		return
	}
	go blCons.loop(ctx)
	return
}

// Close implements io.Closer.
func (blCons *BlockConsumer) Close() error {
	return blCons.bcListener.Close()
}

func (blCons *BlockConsumer) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			blCons.Close()
			return
		case block := <-blCons.ch:
			blCons.consume(ctx, block)
		}
	}
}

func (blCons *BlockConsumer) consume(ctx context.Context, block *types.Block) {
	// Make sure that a block is fully consumed. We don't care about repetitions here.
	for {
		err := blCons.consumeAllTxs(ctx, block)
		if err == nil {
			// Skip temp error check - it should succeed next time
			blCons.blockCounter.SetLatestBlock(block.NumberU64())
			return
		}
		log.Println(err)
		time.Sleep(BlockConsumerBackOff)
	}
}

func (blCons *BlockConsumer) consumeAllTxs(ctx context.Context, block *types.Block) error {
	for _, tx := range block.Transactions() {
		if err := blCons.txHandler.HandleTransaction(ctx, block, tx); err != nil {
			return fmt.Errorf("failed to handle transaction %s: %v", tx.Hash(), err)
		}
	}
	return nil
}
