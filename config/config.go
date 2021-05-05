package config

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/kelseyhightower/envconfig"
)

type envVars struct {
	DBPath                     string `envconfig:"db_path"`
	EthereumRPCEndpoint        string `envconfig:"ethereum_rpc_endpoint"`
	SlackOAuthToken            string `envconfig:"slack_oauth_token"`
	SlackChannelID             string `envconfig:"slack_channel_id"`
	SlackNotifyIntervalSeconds int    `envconfig:"slack_notify_interval_seconds" default:"15"`
	EtherscanBaseURL           string `envconfig:"etherscan_base_url" default:"https://etherscan.io"`

	// Blockchain parameters
	RequireBlockConfirmation uint64 `envconfig:"require_block_confirmation" default:"4"`

	// Large tx detector config
	WatchedTokenAddress   string `envconfig:"watched_token_address"`
	WatchedTokenSymbol    string `envconfig:"watched_token_symbol"`
	WatchedTokenDecimals  int    `envconfig:"watched_token_decimals"`
	WatchedTokenThreshold uint64 `envconfig:"watched_token_threshold"`
}

// Vars are all available config variables in application environment.
var Vars envVars

// Init parses and prepares all config variables.
func Init() {
	override()

	envconfig.MustProcess("", &Vars)
}

// override loads a dev config file to override the environment vars.
// This small feature targets the local development environment.
func override() {
	b, err := ioutil.ReadFile("./devconfig.json")
	if err != nil {
		return
	}

	var configVars map[string]string
	if err := json.Unmarshal(b, &configVars); err != nil {
		panic(err)
	}

	for k, v := range configVars {
		os.Setenv(k, v)
	}
}
