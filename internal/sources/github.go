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

// GitHub checks for GitHub Security Advisories
type GitHub struct {
	notifier notifier.Notifier
	token    string
}

// NewGitHub creates a new GitHub checker
func NewGitHub(n notifier.Notifier, token string) *GitHub {
	return &GitHub{
		notifier: n,
		token:    token,
	}
}

// Check implements the Checker interface
func (c *GitHub) Check(ctx context.Context) (bool, error) {
	log.Println("Checking GitHub Security Advisories...")

	url := fmt.Sprintf("%s?per_page=%d&sort=published&direction=desc",
		config.GitHubSecurityAdvisoriesURL, config.GitHubAPIPerPage)

	data, err := client.FetchGitHubAPI(ctx, url, c.token)
	if err != nil {
		return false, fmt.Errorf("failed to fetch advisories: %w", err)
	}

	var advisories []checker.GitHubAdvisory
	if err := json.Unmarshal(data, &advisories); err != nil {
		return false, fmt.Errorf("failed to parse advisories: %w", err)
	}

	recentAdvisories := c.filterRecentAdvisories(advisories)
	if len(recentAdvisories) == 0 {
		log.Println("No recent security advisories found")
		return false, nil
	}

	message := c.formatMessage(recentAdvisories)
	if err := c.notifier.Notify(ctx, message); err != nil {
		return false, err
	}

	return true, nil
}

func (c *GitHub) filterRecentAdvisories(advisories []checker.GitHubAdvisory) []string {
	var updates []string
	for _, advisory := range advisories {
		if !util.IsRecent(advisory.PublishedAt, config.CheckPeriodHours) {
			continue
		}

		emoji := util.GetSeverityEmoji(advisory.Severity, config.SeverityEmojis)
		summary := util.TruncateText(advisory.Summary, config.MaxAdvisorySummary)
		update := fmt.Sprintf("%s *[%s]* %s\n  <%s|%s>",
			emoji, strings.ToUpper(advisory.Severity), summary, advisory.HTMLURL, advisory.ID)
		updates = append(updates, update)
	}
	return updates
}

func (c *GitHub) formatMessage(updates []string) string {
	return fmt.Sprintf("🔐 *GitHub Security Advisories に新しい脆弱性情報があります！* (%d件)\n\n%s",
		len(updates), strings.Join(updates, "\n\n"))
}
