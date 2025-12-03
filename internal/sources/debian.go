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

// Debian checks for Debian security advisories
type Debian struct {
	notifier notifier.Notifier
}

// NewDebian creates a new Debian checker
func NewDebian(n notifier.Notifier) *Debian {
	return &Debian{notifier: n}
}

// Check implements the Checker interface
func (c *Debian) Check(ctx context.Context) (bool, error) {
	log.Println("Checking Debian Security Advisories...")

	data, err := client.FetchURL(ctx, config.DebianSecurityURL)
	if err != nil {
		return false, fmt.Errorf("failed to fetch Debian security feed: %w", err)
	}

	var feed checker.RSSFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return false, fmt.Errorf("failed to parse Debian RSS feed: %w", err)
	}

	recentAdvisories := c.filterRecentAdvisories(feed.Items)
	if len(recentAdvisories) == 0 {
		log.Println("No recent Debian security advisories found")
		return false, nil
	}

	message := c.formatMessage(recentAdvisories)
	if err := c.notifier.Notify(ctx, message); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Debian) filterRecentAdvisories(items []checker.RSSItem) []string {
	var updates []string
	for _, item := range items {
		if !util.IsRecent(item.Date, config.CheckPeriodHours) {
			continue
		}

		description := util.TruncateText(item.Description, config.MaxAdvisorySummary)
		update := fmt.Sprintf("• *%s*\n  %s\n  <%s|詳細を見る>",
			item.Title, description, item.Link)
		updates = append(updates, update)
	}
	return updates
}

func (c *Debian) formatMessage(updates []string) string {
	return fmt.Sprintf("🐧 *Debian Security Advisories に新しい脆弱性情報があります！* (%d件)\n\n%s",
		len(updates), strings.Join(updates, "\n\n"))
}
