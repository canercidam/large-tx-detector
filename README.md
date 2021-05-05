# Large transaction detector

An extensible service which consumes new Ethereum blocks with different agents.
Current use case is only detecting large transactions and sending Slack channel notifications.

## Building and running locally

```
make run
```

## Running with Docker

First, an `.env` file must be created (gitignored):

```sh
ETHEREUM_RPC_ENDPOINT=<e.g. Infura>
SLACK_OAUTH_TOKEN=<token>
SLACK_CHANNEL_ID=<channel ID>
WATCHED_TOKEN_ADDRESS=0xdac17f958d2ee523a2206206994597c13d831ec7
WATCHED_TOKEN_SYMBOL=USDT
WATCHED_TOKEN_DECIMALS=6
WATCHED_TOKEN_THRESHOLD=1000000
```

and then:

```
make docker-build
make docker
```
