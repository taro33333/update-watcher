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

// GCP checks for GCP release notes updates
type GCP struct {
	notifier notifier.Notifier
}

// NewGCP creates a new GCP checker
func NewGCP(n notifier.Notifier) *GCP {
	return &GCP{notifier: n}
}

// Check implements the Checker interface
func (c *GCP) Check(ctx context.Context) (bool, error) {
	log.Println("Checking GCP Release Notes...")

	data, err := client.FetchURL(ctx, config.GCPReleaseNotesURL)
	if err != nil {
		return false, fmt.Errorf("failed to fetch GCP RSS: %w", err)
	}

	var feed checker.AtomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return false, fmt.Errorf("failed to parse GCP Atom feed: %w", err)
	}

	recentUpdates := c.filterRecentEntries(feed.Entries)
	if len(recentUpdates) == 0 {
		log.Println("No recent GCP updates found")
		return false, nil
	}

	message := c.formatMessage(recentUpdates)
	if err := c.notifier.Notify(ctx, message); err != nil {
		return false, err
	}

	return true, nil
}

func (c *GCP) filterRecentEntries(entries []checker.AtomEntry) []string {
	var updates []string
	for _, entry := range entries {
		dateStr := entry.Updated
		if dateStr == "" {
			dateStr = entry.Published
		}

		if util.IsRecent(dateStr, config.CheckPeriodHours) {
			summary := util.TruncateText(entry.Summary, config.MaxSummaryLength)
			update := fmt.Sprintf("• *%s*\n  %s\n  <%s|詳細を見る>",
				entry.Title, summary, entry.Link.Href)
			updates = append(updates, update)
		}
	}
	return updates
}

func (c *GCP) formatMessage(updates []string) string {
	return fmt.Sprintf("🔥 *GCP Release Notes に新しい更新があります！* (%d件)\n\n%s",
		len(updates), strings.Join(updates, "\n\n"))
}
