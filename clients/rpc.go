package clients

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/canercidam/large-tx-detector/config"
	"github.com/ethereum/go-ethereum"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Config vars
var (
	BlockTime = time.Second * 15
)

// RPC is an Ethereum JSON-RPC client which wraps the go-ethereum client.
type RPC struct {
	*ethclient.Client
	rpc *rpc.Client

	currentBlock uint64
	latestBlock  uint64
	confirmation uint64
	blockCh      chan *types.Block
}

// NewRPC creates a new client.
func NewRPC(ctx context.Context, rawurl string) (*RPC, error) {
	client, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return &RPC{
		Client:       ethclient.NewClient(client),
		rpc:          client,
		confirmation: config.Vars.RequireBlockConfirmation,
	}, nil
}

// Close implements io.Closer.
func (client *RPC) Close() error {
	client.Client.Close()
	client.rpc.Close()
	close(client.blockCh)
	return nil
}

// ListenToNewBlocks listes to the new blocks from the blockchain.
func (client *RPC) ListenToNewBlocks(ctx context.Context, startBlock ...uint64) (ch <-chan *types.Block, err error) {
	if client.currentBlock > 0 {
		return nil, errors.New("only one listener can be started")
	}

	latestBlock, err := client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get the latest block number: %v", err)
	}

	var startAtBlock uint64
	if startBlock != nil {
		startAtBlock = startBlock[0]
	}
	if startAtBlock == 0 {
		startAtBlock = latestBlock - client.confirmation
	}

	client.currentBlock = startAtBlock
	client.latestBlock = latestBlock
	client.blockCh = make(chan *types.Block)
	go client.listenToNewBlocks(ctx)
	return client.blockCh, nil
}

func (client *RPC) shouldProcessNewBlock() bool {
	return client.latestBlock-client.currentBlock >= client.confirmation
}

func (client *RPC) listenToNewBlocks(ctx context.Context) {
	ticker := time.NewTicker(BlockTime)
	for {
		select {
		case <-ctx.Done():
			close(client.blockCh)
			client.Close()
			return
		case <-ticker.C:
			client.latestBlock, _ = client.BlockNumber(ctx)
		default:
			if !client.shouldProcessNewBlock() {
				time.Sleep(BlockTime)
				continue
			}
			block, err := client.BlockByNumber(ctx, big.NewInt(0).SetUint64(client.currentBlock))
			if err == ethereum.NotFound {
				time.Sleep(time.Second * 15)
				continue
			}
			if err != nil {
				log.Printf("failed to get block %d: %v", client.currentBlock, err)
				time.Sleep(time.Second * 5)
				continue
			}
			log.Printf("got new block %d", block.NumberU64())
			client.blockCh <- block // Blocking send
			client.currentBlock++
			time.Sleep(time.Millisecond * 100) // Rate limiting: 10 req/sec
			continue
		}
	}
}

// BatchGetTransactionReceipt does a batch request to get transaction receipts.
func (client *RPC) BatchGetTransactionReceipt(ctx context.Context, txHashes []common.Hash) (receipts []*types.Receipt, err error) {
	if len(txHashes) == 0 {
		return nil, nil
	}

	receipts = make([]*types.Receipt, len(txHashes))
	reqs := make([]rpc.BatchElem, len(txHashes))
	for i, txHash := range txHashes {
		reqs[i] = rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args:   []interface{}{txHash},
			Result: &receipts[i],
		}
	}
	if err := client.rpc.BatchCallContext(ctx, reqs); err != nil {
		return nil, err
	}
	for i, req := range reqs {
		if req.Error != nil {
			return nil, req.Error
		}
		if receipts[i] == nil {
			return nil, fmt.Errorf("got null receipt for tx %s", txHashes[i].Hex())
		}
	}

	return
}
