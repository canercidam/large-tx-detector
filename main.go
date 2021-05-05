package main

import (
	"context"
	"log"

	"github.com/canercidam/large-tx-detector/agents"
	"github.com/canercidam/large-tx-detector/agents/notifier"
	"github.com/canercidam/large-tx-detector/clients"
	"github.com/canercidam/large-tx-detector/core"
	"github.com/canercidam/large-tx-detector/core/agent"
	"github.com/canercidam/large-tx-detector/repository/badgerrepo"

	"github.com/canercidam/large-tx-detector/config"
)

func main() {
	config.Init()

	ctx := context.Background()

	// Initialize the data layer and the clients.
	repo, err := badgerrepo.New(config.Vars.DBPath)
	if err != nil {
		log.Panicf("failed to init the badger repo: %v", err)
	}
	rpcClient, err := clients.NewRPC(ctx, config.Vars.EthereumRPCEndpoint)
	if err != nil {
		log.Panicf("failed to init the rpc client: %v", err)
	}

	// Initialize the agents.
	largeTxDet := agents.NewLargeTxDetector(&agents.LTDConfig{
		AgentID:      "default-agent",
		TokenAddress: config.Vars.WatchedTokenAddress,
		Symbol:       config.Vars.WatchedTokenSymbol,
		Threshold:    config.Vars.WatchedTokenThreshold,
		Notifier:     notifier.NewSlackNotifier(),
		Client:       rpcClient,
	})
	agentPool := agent.NewPool(repo)
	agentPool.AddAgent(largeTxDet)

	// Initialize the consumer, which listes to new blocks and lets agent pool handle.
	blockConsumer := core.NewBlockConsumer(rpcClient, agentPool, repo)
	blockConsumer.Start(ctx)
	<-ctx.Done()
}
