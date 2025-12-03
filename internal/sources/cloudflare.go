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

// Cloudflare checks for Cloudflare Security Blog updates
type Cloudflare struct {
	notifier *notifier.SlackNotifier
}

// NewCloudflare creates a new Cloudflare checker
func NewCloudflare(n *notifier.SlackNotifier) *Cloudflare {
	return &Cloudflare{notifier: n}
}

// Check implements the Checker interface
func (c *Cloudflare) Check(ctx context.Context) (bool, error) {
	log.Println("Checking Cloudflare Security Blog...")

	data, err := client.FetchURL(ctx, config.CloudflareSecurityBlogURL)
	if err != nil {
		return false, fmt.Errorf("failed to fetch Cloudflare security blog: %w", err)
	}

	var feed checker.RSS2Feed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return false, fmt.Errorf("failed to parse Cloudflare RSS feed: %w", err)
	}

	recentPosts := c.filterRecentPosts(feed.Channel.Items)
	if len(recentPosts) == 0 {
		log.Println("No recent Cloudflare security blog posts found")
		return false, nil
	}

	message := c.formatMessage(recentPosts)
	if err := c.notifier.Notify(ctx, message); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Cloudflare) filterRecentPosts(items []checker.RSS2Item) []string {
	var updates []string
	for _, item := range items {
		if !util.IsRecent(item.PubDate, config.CheckPeriodHours) {
			continue
		}

		// Extract plain text from description (remove HTML/CDATA)
		description := c.cleanDescription(item.Description)
		description = util.TruncateText(description, config.MaxSummaryLength)

		update := fmt.Sprintf("• *%s*\n  %s\n  <%s|詳細を見る>",
			item.Title, description, item.Link)
		updates = append(updates, update)
	}
	return updates
}

// cleanDescription removes HTML tags and CDATA from description
func (c *Cloudflare) cleanDescription(html string) string {
	// Remove CDATA markers
	text := strings.ReplaceAll(html, "<![CDATA[", "")
	text = strings.ReplaceAll(text, "]]>", "")

	// Remove common HTML entities
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

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

	return strings.TrimSpace(result.String())
}

func (c *Cloudflare) formatMessage(updates []string) string {
	return fmt.Sprintf("🔶 *Cloudflare Security Blog に新しい記事があります！* (%d件)\n\n%s",
		len(updates), strings.Join(updates, "\n\n"))
}
