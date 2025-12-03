// Package sources provides update checkers for various information sources
// including GCP, Go, Terraform, Debian, and GitHub.
package sources

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"strings"

	"update-watcher/internal/checker"
	"update-watcher/internal/client"
	"update-watcher/internal/config"
	"update-watcher/internal/notifier"
	"update-watcher/internal/util"
)

// AWS checks for AWS Security Bulletins
type AWS struct {
	notifier *notifier.SlackNotifier
}

// NewAWS creates a new AWS checker
func NewAWS(n *notifier.SlackNotifier) *AWS {
	return &AWS{notifier: n}
}

// Check implements the Checker interface
func (c *AWS) Check(ctx context.Context) (bool, error) {
	log.Println("Checking AWS Security Bulletins...")

	data, err := client.FetchURL(ctx, config.AWSSecurityBulletinsURL)
	if err != nil {
		return false, fmt.Errorf("failed to fetch AWS security feed: %w", err)
	}

	var feed checker.RSS2Feed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return false, fmt.Errorf("failed to parse AWS RSS feed: %w", err)
	}

	recentBulletins := c.filterRecentBulletins(feed.Channel.Items)
	if len(recentBulletins) == 0 {
		log.Println("No recent AWS security bulletins found")
		return false, nil
	}

	message := c.formatMessage(recentBulletins)
	if err := c.notifier.Notify(ctx, message); err != nil {
		return false, err
	}

	return true, nil
}

func (c *AWS) filterRecentBulletins(items []checker.RSS2Item) []string {
	var updates []string
	for _, item := range items {
		if !util.IsRecent(item.PubDate, config.CheckPeriodHours) {
			continue
		}

		// Extract bulletin ID from description if available
		description := c.extractPlainText(item.Description)
		description = util.TruncateText(description, config.MaxAdvisorySummary)

		update := fmt.Sprintf("• *%s*\n  %s\n  <%s|詳細を見る>",
			item.Title, description, item.Link)
		updates = append(updates, update)
	}
	return updates
}

// extractPlainText removes HTML tags and extracts plain text from description
func (c *AWS) extractPlainText(html string) string {
	// Simple HTML tag removal
	text := strings.ReplaceAll(html, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")
	text = strings.ReplaceAll(text, "</p>", "\n")

	// Remove HTML tags
	var result strings.Builder
	inTag := false
	for _, char := range text {
		if char == '<' {
			inTag = true
			continue
		}
		if char == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(char)
		}
	}

	// Clean up multiple newlines and spaces
	cleaned := strings.TrimSpace(result.String())
	lines := strings.Split(cleaned, "\n")
	var nonEmpty []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			nonEmpty = append(nonEmpty, line)
		}
	}

	// Return first few lines
	if len(nonEmpty) > 2 {
		return strings.Join(nonEmpty[:2], " ")
	}
	return strings.Join(nonEmpty, " ")
}

func (c *AWS) formatMessage(updates []string) string {
	return fmt.Sprintf("☁️ *AWS Security Bulletins に新しい脆弱性情報があります！* (%d件)\n\n%s",
		len(updates), strings.Join(updates, "\n\n"))
}
