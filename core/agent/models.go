package agent

// Operation is handling of a tx done by an agent.
type Operation struct {
	TxHash      string `json:"txHash"`
	BlockNumber uint64 `json:"blockNumber"`
	AgentID     string `json:"agentId"`
	State       int    `json:"state"`
	Done        bool   `json:"done"`
}
