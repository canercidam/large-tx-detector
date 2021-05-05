package notifier

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/canercidam/large-tx-detector/agents"
	"github.com/canercidam/large-tx-detector/config"
	"github.com/slack-go/slack"
)

// SlackNotifier is a notifier implementation.
type SlackNotifier struct {
	client *slack.Client
	buf    []string
	mu     sync.Mutex
}

// NewSlackNotifier creates a new Slack notifier.a
func NewSlackNotifier() *SlackNotifier {
	sn := &SlackNotifier{client: slack.New(config.Vars.SlackOAuthToken)}
	go sn.loop()
	return sn
}

// Notify notifies a slack channel.
func (sn *SlackNotifier) Notify(ctx context.Context, notif *agents.LargeTxNotification) error {
	sn.mu.Lock()
	defer sn.mu.Unlock()
	sn.buf = append(sn.buf, fmt.Sprintf(
		`*Tx:* <%s/tx/%s|%s>
*From:* %s
*To:* %s
*Amount:* %.2f %s`,
		config.Vars.EtherscanBaseURL, notif.Hash, notif.Hash, notif.From, notif.To, notif.Value, notif.Symbol,
	))
	return nil
}

func (sn *SlackNotifier) loop() {
	ticker := time.NewTicker(time.Second * (time.Duration)(config.Vars.SlackNotifyIntervalSeconds))
	for _ = range ticker.C {
		sn.postBufferedMessages()
	}
}

func (sn *SlackNotifier) postBufferedMessages() {
	sn.mu.Lock()
	defer sn.mu.Unlock()
	if len(sn.buf) == 0 {
		return
	}
	_, _, err := sn.client.PostMessage(config.Vars.SlackChannelID, slack.MsgOptionText(
		strings.Join(sn.buf, "\n\n"), false,
	))
	if err != nil {
		log.Printf("failed to post the slack message: %v", err)
		return
	}
	sn.buf = nil
}
