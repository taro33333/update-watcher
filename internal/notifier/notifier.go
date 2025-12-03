// Package notifier provides notification functionality for sending
// messages to external services such as Slack or stdout.
package notifier

import "context"

// UpdateInfo represents a structured update notification
type UpdateInfo struct {
	URL     string `json:"url"`
	Project string `json:"project"`
	Version string `json:"version,omitempty"`
	Title   string `json:"title,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// Notifier defines the interface for sending notifications
type Notifier interface {
	// Notify sends a formatted message (for Slack-style notifications)
	Notify(ctx context.Context, message string) error

	// NotifyUpdate sends structured update information (for JSON output)
	NotifyUpdate(ctx context.Context, info UpdateInfo) error
}

