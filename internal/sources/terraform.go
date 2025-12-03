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

// Terraform checks for Terraform releases
type Terraform struct {
	notifier notifier.Notifier
	token    string
}

// NewTerraform creates a new Terraform checker
func NewTerraform(n notifier.Notifier, token string) *Terraform {
	return &Terraform{
		notifier: n,
		token:    token,
	}
}

// Check implements the Checker interface
func (c *Terraform) Check(ctx context.Context) (bool, error) {
	log.Println("Checking Terraform Releases...")

	data, err := client.FetchGitHubAPI(ctx, config.TerraformReleasesURL, c.token)
	if err != nil {
		return false, fmt.Errorf("failed to fetch Terraform releases: %w", err)
	}

	var releases []checker.GitHubRelease
	if err := json.Unmarshal(data, &releases); err != nil {
		return false, fmt.Errorf("failed to parse Terraform releases: %w", err)
	}

	recentReleases := c.filterRecentReleases(releases)
	if len(recentReleases) == 0 {
		log.Println("No recent Terraform releases found")
		return false, nil
	}

	message := c.formatMessage(recentReleases)
	if err := c.notifier.Notify(ctx, message); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Terraform) filterRecentReleases(releases []checker.GitHubRelease) []string {
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

func (c *Terraform) formatMessage(updates []string) string {
	return fmt.Sprintf("🏗️ *Terraform に新しいリリースがあります！* (%d件)\n\n%s",
		len(updates), strings.Join(updates, "\n\n"))
}
