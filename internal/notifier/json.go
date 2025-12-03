package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sync"
)

// JSONNotifier outputs update information as JSON to stdout
type JSONNotifier struct {
	writer io.Writer
	mu     sync.Mutex
}

// NewJSON creates a new JSONNotifier that writes to stdout
func NewJSON() *JSONNotifier {
	return &JSONNotifier{
		writer: os.Stdout,
	}
}

// NewJSONWithWriter creates a new JSONNotifier with a custom writer
func NewJSONWithWriter(w io.Writer) *JSONNotifier {
	return &JSONNotifier{
		writer: w,
	}
}

// Notify parses a Slack-formatted message and outputs as JSON
// This is for backward compatibility with existing sources
func (j *JSONNotifier) Notify(ctx context.Context, message string) error {
	// Extract URLs from the message
	urls := extractURLs(message)
	for _, url := range urls {
		info := UpdateInfo{
			URL:     url,
			Title:   extractTitle(message),
			Summary: message,
		}
		if err := j.NotifyUpdate(ctx, info); err != nil {
			return err
		}
	}
	return nil
}

// NotifyUpdate outputs structured update information as JSON
func (j *JSONNotifier) NotifyUpdate(ctx context.Context, info UpdateInfo) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal update info: %w", err)
	}

	_, err = fmt.Fprintln(j.writer, string(data))
	return err
}

// extractURLs extracts all URLs from a message
func extractURLs(message string) []string {
	// Match URLs in Slack format <URL|text> or plain URLs
	slackURLRegex := regexp.MustCompile(`<(https?://[^|>]+)(?:\|[^>]*)?>`)
	plainURLRegex := regexp.MustCompile(`https?://[^\s<>]+`)

	var urls []string
	seen := make(map[string]bool)

	// First, extract Slack-formatted URLs
	matches := slackURLRegex.FindAllStringSubmatch(message, -1)
	for _, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			urls = append(urls, match[1])
			seen[match[1]] = true
		}
	}

	// Then, extract plain URLs
	plainMatches := plainURLRegex.FindAllString(message, -1)
	for _, url := range plainMatches {
		if !seen[url] {
			urls = append(urls, url)
			seen[url] = true
		}
	}

	return urls
}

// extractTitle extracts the title from a Slack-formatted message
func extractTitle(message string) string {
	// Match *title* pattern (Slack bold)
	titleRegex := regexp.MustCompile(`\*([^*]+)\*`)
	matches := titleRegex.FindStringSubmatch(message)
	if len(matches) > 1 {
		return matches[1]
	}
	// Return first line as fallback
	for i, c := range message {
		if c == '\n' {
			return message[:i]
		}
	}
	if len(message) > 100 {
		return message[:100]
	}
	return message
}

