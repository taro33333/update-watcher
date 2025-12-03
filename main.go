package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	gcpReleaseNotesRSS          = "https://cloud.google.com/feeds/release-notes.xml"
	goReleasesRSS               = "https://go.dev/doc/devel/release.rss"
	githubSecurityAdvisoriesAPI = "https://api.github.com/advisories"
	checkPeriodHours            = 25 // 過去25時間の更新をチェック（1日1回実行なので余裕を持たせる）
)

// RSS Feed structures
type RSS struct {
	Channel Channel `xml:"channel"`
}

type Channel struct {
	Items []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

// AtomFeed represents an Atom feed structure (GCP uses Atom format)
type AtomFeed struct {
	Entries []AtomEntry `xml:"entry"`
}

type AtomEntry struct {
	Title     string   `xml:"title"`
	Link      AtomLink `xml:"link"`
	Published string   `xml:"published"`
	Updated   string   `xml:"updated"`
	Summary   string   `xml:"summary"`
}

type AtomLink struct {
	Href string `xml:"href,attr"`
}

// GitHubAdvisory represents a GitHub Security Advisory
type GitHubAdvisory struct {
	ID          string `json:"ghsa_id"`
	Summary     string `json:"summary"`
	Severity    string `json:"severity"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
}

type SlackMessage struct {
	Text string `json:"text"`
}

func notifySlack(msg string) error {
	webhook := os.Getenv("SLACK_WEBHOOK_URL")
	if webhook == "" {
		return fmt.Errorf("SLACK_WEBHOOK_URL not set")
	}

	body, err := json.Marshal(SlackMessage{Text: msg})
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to post to slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

func fetchURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return b, nil
}

func isRecent(dateStr string, hoursAgo int) bool {
	// Try multiple date formats
	formats := []string{
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		"2006-01-02T15:04:05Z",
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 MST",
	}

	var parsedTime time.Time
	var err error

	for _, format := range formats {
		parsedTime, err = time.Parse(format, dateStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Printf("Warning: failed to parse date '%s': %v", dateStr, err)
		return false
	}

	cutoff := time.Now().Add(-time.Duration(hoursAgo) * time.Hour)
	return parsedTime.After(cutoff)
}

func checkGCPReleaseNotes() (bool, error) {
	log.Println("Checking GCP Release Notes...")
	data, err := fetchURL(gcpReleaseNotesRSS)
	if err != nil {
		return false, fmt.Errorf("failed to fetch GCP RSS: %w", err)
	}

	var feed AtomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return false, fmt.Errorf("failed to parse GCP Atom feed: %w", err)
	}

	var recentUpdates []string
	for _, entry := range feed.Entries {
		dateStr := entry.Published
		if entry.Updated != "" {
			dateStr = entry.Updated
		}

		if isRecent(dateStr, checkPeriodHours) {
			summary := strings.TrimSpace(entry.Summary)
			if len(summary) > 200 {
				summary = summary[:200] + "..."
			}
			recentUpdates = append(recentUpdates, fmt.Sprintf("• *%s*\n  %s\n  <%s|詳細を見る>",
				entry.Title, summary, entry.Link.Href))
		}
	}

	if len(recentUpdates) > 0 {
		msg := fmt.Sprintf("🔥 *GCP Release Notes に新しい更新があります！* (%d件)\n\n%s",
			len(recentUpdates), strings.Join(recentUpdates, "\n\n"))
		if err := notifySlack(msg); err != nil {
			return false, err
		}
		return true, nil
	}

	log.Println("No recent GCP updates found")
	return false, nil
}

func checkGoReleases() (bool, error) {
	log.Println("Checking Go Releases...")
	data, err := fetchURL(goReleasesRSS)
	if err != nil {
		return false, fmt.Errorf("failed to fetch Go RSS: %w", err)
	}

	var rss RSS
	if err := xml.Unmarshal(data, &rss); err != nil {
		return false, fmt.Errorf("failed to parse Go RSS: %w", err)
	}

	var recentReleases []string
	for _, item := range rss.Channel.Items {
		if isRecent(item.PubDate, checkPeriodHours) {
			desc := strings.TrimSpace(item.Description)
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			recentReleases = append(recentReleases, fmt.Sprintf("• *%s*\n  %s\n  <%s|詳細を見る>",
				item.Title, desc, item.Link))
		}
	}

	if len(recentReleases) > 0 {
		msg := fmt.Sprintf("🦫 *Go に新しいリリースがあります！* (%d件)\n\n%s",
			len(recentReleases), strings.Join(recentReleases, "\n\n"))
		if err := notifySlack(msg); err != nil {
			return false, err
		}
		return true, nil
	}

	log.Println("No recent Go releases found")
	return false, nil
}

func checkGitHubSecurityAdvisories() (bool, error) {
	log.Println("Checking GitHub Security Advisories...")

	// GitHub Token (optional but recommended for rate limits)
	token := os.Getenv("GITHUB_TOKEN")

	// Get advisories from the last day
	url := fmt.Sprintf("%s?per_page=100&sort=published&direction=desc", githubSecurityAdvisoriesAPI)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to fetch advisories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var advisories []GitHubAdvisory
	if err := json.NewDecoder(resp.Body).Decode(&advisories); err != nil {
		return false, fmt.Errorf("failed to parse advisories: %w", err)
	}

	var recentAdvisories []string
	for _, advisory := range advisories {
		if isRecent(advisory.PublishedAt, checkPeriodHours) {
			severityEmoji := "⚠️"
			switch strings.ToLower(advisory.Severity) {
			case "critical":
				severityEmoji = "🚨"
			case "high":
				severityEmoji = "❗"
			case "medium":
				severityEmoji = "⚠️"
			case "low":
				severityEmoji = "ℹ️"
			}

			summary := strings.TrimSpace(advisory.Summary)
			if len(summary) > 150 {
				summary = summary[:150] + "..."
			}

			recentAdvisories = append(recentAdvisories, fmt.Sprintf("%s *[%s]* %s\n  <%s|%s>",
				severityEmoji, strings.ToUpper(advisory.Severity), summary, advisory.HTMLURL, advisory.ID))
		}
	}

	if len(recentAdvisories) > 0 {
		msg := fmt.Sprintf("🔐 *GitHub Security Advisories に新しい脆弱性情報があります！* (%d件)\n\n%s",
			len(recentAdvisories), strings.Join(recentAdvisories, "\n\n"))
		if err := notifySlack(msg); err != nil {
			return false, err
		}
		return true, nil
	}

	log.Println("No recent security advisories found")
	return false, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting update watcher...")

	now := time.Now().Format("2006-01-02 15:04:05 MST")

	// Check if Slack webhook is configured
	if os.Getenv("SLACK_WEBHOOK_URL") == "" {
		log.Fatal("SLACK_WEBHOOK_URL environment variable is not set")
	}

	var errors []string
	hasUpdates := false

	// 1. Check GCP Release Notes
	if updated, err := checkGCPReleaseNotes(); err != nil {
		log.Printf("Error checking GCP: %v", err)
		errors = append(errors, fmt.Sprintf("❌ GCP Release Notes: %v", err))
	} else if updated {
		hasUpdates = true
	}

	// 2. Check Go Releases
	if updated, err := checkGoReleases(); err != nil {
		log.Printf("Error checking Go: %v", err)
		errors = append(errors, fmt.Sprintf("❌ Go Releases: %v", err))
	} else if updated {
		hasUpdates = true
	}

	// 3. Check GitHub Security Advisories
	if updated, err := checkGitHubSecurityAdvisories(); err != nil {
		log.Printf("Error checking GitHub: %v", err)
		errors = append(errors, fmt.Sprintf("❌ GitHub Security Advisories: %v", err))
	} else if updated {
		hasUpdates = true
	}

	// Send summary message
	if len(errors) > 0 {
		errMsg := fmt.Sprintf("⚠️ *アップデート監視完了（一部エラーあり）* - %s\n\n%s",
			now, strings.Join(errors, "\n"))
		if err := notifySlack(errMsg); err != nil {
			log.Printf("Failed to send error notification: %v", err)
		}
	} else if !hasUpdates {
		summaryMsg := fmt.Sprintf("✅ *アップデート監視完了* - %s\n新しい更新はありませんでした。",
			now)
		if err := notifySlack(summaryMsg); err != nil {
			log.Printf("Failed to send summary: %v", err)
		}
	} else {
		summaryMsg := fmt.Sprintf("✅ *アップデート監視完了* - %s\n上記の更新を確認してください。",
			now)
		if err := notifySlack(summaryMsg); err != nil {
			log.Printf("Failed to send summary: %v", err)
		}
	}

	log.Println("Update watcher completed successfully")
}
