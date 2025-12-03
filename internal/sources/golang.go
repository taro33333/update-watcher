package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"update-watcher/internal/checker"
	"update-watcher/internal/client"
	"update-watcher/internal/config"
	"update-watcher/internal/notifier"
	"update-watcher/internal/util"
)

// Go checks for Go releases updates
type Go struct {
	notifier notifier.Notifier
	token    string
}

// NewGo creates a new Go checker
func NewGo(n notifier.Notifier, token string) *Go {
	return &Go{
		notifier: n,
		token:    token,
	}
}

// Check implements the Checker interface
func (c *Go) Check(ctx context.Context) (bool, error) {
	log.Println("Checking Go Releases...")

	data, err := client.FetchGitHubAPI(ctx, config.GoReleasesURL, c.token)
	if err != nil {
		return false, fmt.Errorf("failed to fetch Go releases: %w", err)
	}

	var releases []checker.GitHubRelease
	if err := json.Unmarshal(data, &releases); err != nil {
		return false, fmt.Errorf("failed to parse Go releases: %w", err)
	}

	recentReleases := c.filterRecentReleases(releases)
	if len(recentReleases) == 0 {
		log.Println("No recent Go releases found")
		return false, nil
	}

	message := c.formatMessage(recentReleases)
	if err := c.notifier.Notify(ctx, message); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Go) filterRecentReleases(releases []checker.GitHubRelease) []string {
	var updates []string
	for _, release := range releases {
		if !util.IsRecent(release.PublishedAt, config.CheckPeriodHours) {
			continue
		}

		title := release.Name
		if title == "" {
			title = release.TagName
		}

		body := util.FormatMultilineText(release.Body, config.MaxSummaryLength, config.MaxDescriptionLines)
		update := fmt.Sprintf("• *%s*\n  %s\n  <%s|詳細を見る>",
			title, body, release.HTMLURL)
		updates = append(updates, update)
	}
	return updates
}

func (c *Go) formatMessage(updates []string) string {
	return fmt.Sprintf("🦫 *Go に新しいリリースがあります！* (%d件)\n\n%s",
		len(updates), strings.Join(updates, "\n\n"))
}
