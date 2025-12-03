package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"update-watcher/internal/client"
)

// SlackMessage represents a Slack webhook message
type SlackMessage struct {
	Text string `json:"text"`
}

// SlackNotifier handles Slack notifications
type SlackNotifier struct {
	webhookURL string
}

// NewSlack creates a new SlackNotifier
func NewSlack(webhookURL string) *SlackNotifier {
	return &SlackNotifier{webhookURL: webhookURL}
}

// New creates a new SlackNotifier (alias for backward compatibility)
func New(webhookURL string) *SlackNotifier {
	return NewSlack(webhookURL)
}

// Notify sends a message to Slack
func (s *SlackNotifier) Notify(ctx context.Context, message string) error {
	body, err := json.Marshal(SlackMessage{Text: message})
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post to slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

// NotifyUpdate sends structured update information to Slack
// For Slack, we just format it as a message
func (s *SlackNotifier) NotifyUpdate(ctx context.Context, info UpdateInfo) error {
	message := fmt.Sprintf("*%s*", info.Title)
	if info.Version != "" {
		message += fmt.Sprintf(" (v%s)", info.Version)
	}
	message += fmt.Sprintf("\n%s\n<%s|詳細を見る>", info.Summary, info.URL)
	return s.Notify(ctx, message)
}
