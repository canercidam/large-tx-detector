package agents

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/canercidam/large-tx-detector/clients"
	"github.com/canercidam/large-tx-detector/config"
	"github.com/canercidam/large-tx-detector/contracts"

	"github.com/canercidam/large-tx-detector/core/agent"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	transferTopicHash = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
)

// LargeTxNotification contains the notification data.
type LargeTxNotification struct {
	Hash   string
	From   string
	To     string
	Value  float64
	Symbol string
}

// LargeTxNotifier sends notifications about large txs.
type LargeTxNotifier interface {
	Notify(context.Context, *LargeTxNotification) error
}

type defaultLTNotifier struct{}

func (dltn *defaultLTNotifier) Notify(ctx context.Context, notif *LargeTxNotification) error {
	log.Printf(
		"notification: large tx %s detected from %s to %s of amount %.5f %s",
		notif.Hash, notif.From, notif.To, notif.Value, notif.Symbol,
	)
	return nil
}

// LTDConfig contains the large tx detector agent config parameters.
type LTDConfig struct {
	AgentID      string
	TokenAddress string
	Symbol       string
	Threshold    uint64
	Notifier     LargeTxNotifier
	Client       *clients.RPC
}

// LargeTxDetector detects the large transactions and implements the agent.Agent interface.
type LargeTxDetector struct {
	config       *LTDConfig
	tokenAddress common.Address
	decimals     int
	exp          *big.Int
	threshold    *big.Int
	notifier     LargeTxNotifier
	client       *clients.RPC
	contract     *bind.BoundContract

	currentOp       *agent.Operation
	currentTx       *types.Transaction
	currentBlock    uint64
	currentReceipts []*types.Receipt
	currentState    int
}

// NewLargeTxDetector creates a new large tx detector.
func NewLargeTxDetector(conf *LTDConfig) *LargeTxDetector {
	ltd := &LargeTxDetector{config: conf}
	ltd.tokenAddress = common.HexToAddress(conf.TokenAddress)
	ltd.decimals = config.Vars.WatchedTokenDecimals
	ltd.notifier = conf.Notifier
	ltd.client = conf.Client
	ltd.contract, _ = contracts.BindIERC20(ltd.tokenAddress, nil, nil, nil)

	// Convert float amount to wei.
	ltd.exp = big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(ltd.decimals)), nil)
	ltd.threshold = big.NewInt(0).Mul(big.NewInt(0).SetUint64(conf.Threshold), ltd.exp)

	// Use default log notifier if a notifier was not specified.
	if ltd.notifier == nil {
		ltd.notifier = &defaultLTNotifier{}
	}

	return ltd
}

// Skip checks the logs bloom filter to see if we should skip this block entirely.
func (ltd *LargeTxDetector) Skip(block *types.Block, tx *types.Transaction) bool {
	hasTokenAddress := block.Bloom().Test(ltd.tokenAddress.Bytes())
	hasTopic := block.Bloom().Test(transferTopicHash.Bytes())
	return !(hasTokenAddress && hasTopic)
}

// ID returns the agent ID.
func (ltd *LargeTxDetector) ID() string {
	return ltd.config.AgentID
}

// Init inits the tx handling.
func (ltd *LargeTxDetector) Init(op *agent.Operation, tx *types.Transaction) {
	ltd.currentOp = op
	ltd.currentTx = tx
	ltd.currentState = op.State
}

// Next tells if we have a next state to continue handling.
// We only check something and send a notification so we should have
// the initial and the final state only.
func (ltd *LargeTxDetector) Next() bool {
	ltd.currentState++
	return ltd.currentState < 2
}

// HandleTransaction handles a transaction using the block info.
func (ltd *LargeTxDetector) HandleTransaction(ctx context.Context, block *types.Block, tx *types.Transaction) error {
	if err := ltd.ensureTxLogs(ctx, block); err != nil {
		return err
	}

	transferLog, ok := ltd.findTransferLog(tx)
	if !ok {
		return nil
	}

	event, err := contracts.UnpackIERC20Transfer(ltd.contract, transferLog)
	if err != nil {
		return fmt.Errorf("failed to unpack the event: %v", err)
	}

	if event.Value.Cmp(ltd.threshold) < 0 {
		return nil
	}

	return ltd.notifier.Notify(ctx, &LargeTxNotification{
		Hash:   tx.Hash().Hex(),
		From:   event.From.Hex(),
		To:     event.To.Hex(),
		Value:  ltd.readableAmount(event.Value),
		Symbol: ltd.config.Symbol,
	})
}

// ensureTxLogs ensures that we have the tx logs for the newest block.
func (ltd *LargeTxDetector) ensureTxLogs(ctx context.Context, block *types.Block) error {
	currentBlock := block.NumberU64()
	if currentBlock == ltd.currentBlock {
		return nil
	}
	var txHashes []common.Hash
	for _, tx := range block.Transactions() {
		txHashes = append(txHashes, tx.Hash())
	}
	receipts, err := ltd.client.BatchGetTransactionReceipt(ctx, txHashes)
	if err != nil {
		return fmt.Errorf("failed to get the transaction logs for block %d: %v", currentBlock, err)
	}
	ltd.currentReceipts = receipts
	ltd.currentBlock = currentBlock
	return nil
}

func (ltd *LargeTxDetector) findTransferLog(tx *types.Transaction) (*types.Log, bool) {
	for _, receipt := range ltd.currentReceipts {
		if receipt.TxHash.Hex() != tx.Hash().Hex() {
			continue
		}
		for _, txLog := range receipt.Logs {
			if txLog.Address.String() != ltd.tokenAddress.String() {
				return nil, false
			}
			for _, topicHash := range txLog.Topics {
				if topicHash.Hex() == transferTopicHash.Hex() {
					return txLog, true
				}
			}
		}
	}
	return nil, false
}

func (ltd *LargeTxDetector) readableAmount(realAmount *big.Int) float64 {
	return float64(big.NewInt(0).Div(realAmount, ltd.exp).Uint64())
}
