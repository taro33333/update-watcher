// Package sources provides update checkers for various information sources
// including GCP, Go, Terraform, Debian, and GitHub.
package sources

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"regexp"
	"strings"

	"update-watcher/internal/checker"
	"update-watcher/internal/client"
	"update-watcher/internal/config"
	"update-watcher/internal/notifier"
	"update-watcher/internal/util"
)

// GCPSecurity checks for GCP Security Bulletins
type GCPSecurity struct {
	notifier notifier.Notifier
}

// NewGCPSecurity creates a new GCP Security checker
func NewGCPSecurity(n notifier.Notifier) *GCPSecurity {
	return &GCPSecurity{notifier: n}
}

// Check implements the Checker interface
func (c *GCPSecurity) Check(ctx context.Context) (bool, error) {
	log.Println("Checking GCP Security Bulletins...")

	data, err := client.FetchURL(ctx, config.GCPSecurityBulletinsURL)
	if err != nil {
		return false, fmt.Errorf("failed to fetch GCP security feed: %w", err)
	}

	var feed checker.AtomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return false, fmt.Errorf("failed to parse GCP security Atom feed: %w", err)
	}

	recentBulletins := c.filterRecentBulletins(feed.Entries)
	if len(recentBulletins) == 0 {
		log.Println("No recent GCP security bulletins found")
		return false, nil
	}

	message := c.formatMessage(recentBulletins)
	if err := c.notifier.Notify(ctx, message); err != nil {
		return false, err
	}

	return true, nil
}

func (c *GCPSecurity) filterRecentBulletins(entries []checker.AtomEntry) []string {
	var updates []string
	for _, entry := range entries {
		dateStr := entry.Updated
		if dateStr == "" {
			dateStr = entry.Published
		}

		if util.IsRecent(dateStr, config.CheckPeriodHours) {
			// Extract CVE IDs and severity from HTML content
			cveIDs := c.extractCVEIDs(entry.Summary)
			severity := c.extractSeverity(entry.Summary)

			// Clean HTML from summary
			summary := c.cleanHTMLContent(entry.Summary)
			summary = util.TruncateText(summary, config.MaxAdvisorySummary)

			// Format bulletin info
			cveInfo := ""
			if len(cveIDs) > 0 {
				cveInfo = fmt.Sprintf(" [%s]", strings.Join(cveIDs, ", "))
			}

			severityEmoji := c.getSeverityEmoji(severity)

			update := fmt.Sprintf("%s *%s*%s\n  %s\n  <%s|詳細を見る>",
				severityEmoji, entry.Title, cveInfo, summary, entry.Link.Href)
			updates = append(updates, update)
		}
	}
	return updates
}

// extractCVEIDs extracts CVE IDs from HTML content
func (c *GCPSecurity) extractCVEIDs(html string) []string {
	// Match CVE-YYYY-NNNNN pattern
	re := regexp.MustCompile(`CVE-\d{4}-\d{4,7}`)
	matches := re.FindAllString(html, -1)

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, cve := range matches {
		if !seen[cve] {
			seen[cve] = true
			unique = append(unique, cve)
		}
	}

	// Limit to first 3 CVEs for readability
	if len(unique) > 3 {
		return unique[:3]
	}
	return unique
}

// extractSeverity extracts severity level from HTML content
func (c *GCPSecurity) extractSeverity(html string) string {
	html = strings.ToLower(html)
	severities := []string{"critical", "high", "moderate", "medium", "low"}

	for _, severity := range severities {
		if strings.Contains(html, ">"+severity+"<") || strings.Contains(html, ">"+severity+"</td>") {
			return severity
		}
	}
	return "unknown"
}

// cleanHTMLContent removes HTML tags from content
func (c *GCPSecurity) cleanHTMLContent(html string) string {
	// Remove CDATA
	text := strings.ReplaceAll(html, "<![CDATA[", "")
	text = strings.ReplaceAll(text, "]]>", "")

	// Remove HTML entities
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")

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

	// Clean up whitespace
	cleaned := strings.TrimSpace(result.String())
	// Replace multiple spaces/newlines with single space
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")

	return cleaned
}

// getSeverityEmoji returns emoji for severity level
func (c *GCPSecurity) getSeverityEmoji(severity string) string {
	emojiMap := map[string]string{
		"critical": "🚨",
		"high":     "❗",
		"moderate": "⚠️",
		"medium":   "⚠️",
		"low":      "ℹ️",
	}

	if emoji, ok := emojiMap[strings.ToLower(severity)]; ok {
		return emoji
	}
	return "🔷"
}

func (c *GCPSecurity) formatMessage(updates []string) string {
	return fmt.Sprintf("🔷 *GCP Security Bulletins に新しい脆弱性情報があります！* (%d件)\n\n%s",
		len(updates), strings.Join(updates, "\n\n"))
}
