package main

import (
	"bytes"
	"context"
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

// Configuration constants
const (
	gcpReleaseNotesURL          = "https://cloud.google.com/feeds/gcp-release-notes.xml"
	goReleasesURL               = "https://api.github.com/repos/golang/go/releases"
	githubSecurityAdvisoriesURL = "https://api.github.com/advisories"

	checkPeriodHours    = 25 // 過去25時間の更新をチェック（1日1回実行なので余裕を持たせる）
	httpTimeout         = 30 * time.Second
	maxSummaryLength    = 200
	maxAdvisorySummary  = 150
	maxDescriptionLines = 3
	githubAPIPerPage    = 100
)

// Severity emoji mapping
var severityEmojis = map[string]string{
	"critical": "🚨",
	"high":     "❗",
	"medium":   "⚠️",
	"low":      "ℹ️",
}

// Global HTTP client (reused for better performance)
var httpClient = &http.Client{
	Timeout: httpTimeout,
}

// ========================================
// Domain Models
// ========================================

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

// GitHubRelease represents a GitHub Release
type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
}

// GitHubAdvisory represents a GitHub Security Advisory
type GitHubAdvisory struct {
	ID          string `json:"ghsa_id"`
	Summary     string `json:"summary"`
	Severity    string `json:"severity"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
}

// SlackMessage represents a Slack webhook message
type SlackMessage struct {
	Text string `json:"text"`
}

// UpdateResult represents the result of an update check
type UpdateResult struct {
	HasUpdates bool
	Error      error
}

// ========================================
// HTTP Client Utilities
// ========================================

// fetchURL performs a simple HTTP GET request
func fetchURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}

// fetchGitHubAPI performs a GitHub API request with authentication
func fetchGitHubAPI(ctx context.Context, url, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GitHub API: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(data))
	}

	return data, nil
}

// ========================================
// Date Parsing Utilities
// ========================================

var supportedDateFormats = []string{
	time.RFC3339,
	time.RFC1123Z,
	time.RFC1123,
	"2006-01-02T15:04:05Z",
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"Mon, 02 Jan 2006 15:04:05 MST",
}

// parseDate attempts to parse a date string using multiple formats
func parseDate(dateStr string) (time.Time, error) {
	for _, format := range supportedDateFormats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse date: %s", dateStr)
}

// isRecent checks if a date is within the specified hours ago
func isRecent(dateStr string, hoursAgo int) bool {
	parsedTime, err := parseDate(dateStr)
	if err != nil {
		log.Printf("Warning: %v", err)
		return false
	}

	cutoff := time.Now().Add(-time.Duration(hoursAgo) * time.Hour)
	return parsedTime.After(cutoff)
}

// ========================================
// Text Formatting Utilities
// ========================================

// truncateText truncates text to maxLength and adds ellipsis
func truncateText(text string, maxLength int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}

// formatMultilineText formats multiline text, limiting to maxLines
func formatMultilineText(text string, maxLength, maxLines int) string {
	text = strings.TrimSpace(text)
	if len(text) > maxLength {
		text = text[:maxLength] + "..."
	}

	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	if len(lines) > maxLines {
		return strings.Join(lines[:maxLines], "\n") + "..."
	}

	return text
}

// getSeverityEmoji returns the emoji for a given severity level
func getSeverityEmoji(severity string) string {
	if emoji, ok := severityEmojis[strings.ToLower(severity)]; ok {
		return emoji
	}
	return "⚠️" // default
}

// ========================================
// Slack Notification
// ========================================

// SlackNotifier handles Slack notifications
type SlackNotifier struct {
	webhookURL string
}

// NewSlackNotifier creates a new SlackNotifier
func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{webhookURL: webhookURL}
}

// Notify sends a message to Slack
func (s *SlackNotifier) Notify(ctx context.Context, message string) error {
	body, err := json.Marshal(SlackMessage{Text: message})
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post to slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}

// ========================================
// Update Checkers
// ========================================

// GCPReleaseChecker checks for GCP release notes updates
type GCPReleaseChecker struct {
	notifier *SlackNotifier
}

func NewGCPReleaseChecker(notifier *SlackNotifier) *GCPReleaseChecker {
	return &GCPReleaseChecker{notifier: notifier}
}

func (c *GCPReleaseChecker) Check(ctx context.Context) (bool, error) {
	log.Println("Checking GCP Release Notes...")

	data, err := fetchURL(ctx, gcpReleaseNotesURL)
	if err != nil {
		return false, fmt.Errorf("failed to fetch GCP RSS: %w", err)
	}

	var feed AtomFeed
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

func (c *GCPReleaseChecker) filterRecentEntries(entries []AtomEntry) []string {
	var updates []string
	for _, entry := range entries {
		dateStr := entry.Updated
		if dateStr == "" {
			dateStr = entry.Published
		}

		if isRecent(dateStr, checkPeriodHours) {
			summary := truncateText(entry.Summary, maxSummaryLength)
			update := fmt.Sprintf("• *%s*\n  %s\n  <%s|詳細を見る>",
				entry.Title, summary, entry.Link.Href)
			updates = append(updates, update)
		}
	}
	return updates
}

func (c *GCPReleaseChecker) formatMessage(updates []string) string {
	return fmt.Sprintf("🔥 *GCP Release Notes に新しい更新があります！* (%d件)\n\n%s",
		len(updates), strings.Join(updates, "\n\n"))
}

// GoReleaseChecker checks for Go releases updates
type GoReleaseChecker struct {
	notifier *SlackNotifier
	token    string
}

func NewGoReleaseChecker(notifier *SlackNotifier, token string) *GoReleaseChecker {
	return &GoReleaseChecker{
		notifier: notifier,
		token:    token,
	}
}

func (c *GoReleaseChecker) Check(ctx context.Context) (bool, error) {
	log.Println("Checking Go Releases...")

	data, err := fetchGitHubAPI(ctx, goReleasesURL, c.token)
	if err != nil {
		return false, fmt.Errorf("failed to fetch Go releases: %w", err)
	}

	var releases []GitHubRelease
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

func (c *GoReleaseChecker) filterRecentReleases(releases []GitHubRelease) []string {
	var updates []string
	for _, release := range releases {
		if !isRecent(release.PublishedAt, checkPeriodHours) {
			continue
		}

		title := release.Name
		if title == "" {
			title = release.TagName
		}

		body := formatMultilineText(release.Body, maxSummaryLength, maxDescriptionLines)
		update := fmt.Sprintf("• *%s*\n  %s\n  <%s|詳細を見る>",
			title, body, release.HTMLURL)
		updates = append(updates, update)
	}
	return updates
}

func (c *GoReleaseChecker) formatMessage(updates []string) string {
	return fmt.Sprintf("🦫 *Go に新しいリリースがあります！* (%d件)\n\n%s",
		len(updates), strings.Join(updates, "\n\n"))
}

// SecurityAdvisoryChecker checks for GitHub Security Advisories
type SecurityAdvisoryChecker struct {
	notifier *SlackNotifier
	token    string
}

func NewSecurityAdvisoryChecker(notifier *SlackNotifier, token string) *SecurityAdvisoryChecker {
	return &SecurityAdvisoryChecker{
		notifier: notifier,
		token:    token,
	}
}

func (c *SecurityAdvisoryChecker) Check(ctx context.Context) (bool, error) {
	log.Println("Checking GitHub Security Advisories...")

	url := fmt.Sprintf("%s?per_page=%d&sort=published&direction=desc",
		githubSecurityAdvisoriesURL, githubAPIPerPage)

	data, err := fetchGitHubAPI(ctx, url, c.token)
	if err != nil {
		return false, fmt.Errorf("failed to fetch advisories: %w", err)
	}

	var advisories []GitHubAdvisory
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

func (c *SecurityAdvisoryChecker) filterRecentAdvisories(advisories []GitHubAdvisory) []string {
	var updates []string
	for _, advisory := range advisories {
		if !isRecent(advisory.PublishedAt, checkPeriodHours) {
			continue
		}

		emoji := getSeverityEmoji(advisory.Severity)
		summary := truncateText(advisory.Summary, maxAdvisorySummary)
		update := fmt.Sprintf("%s *[%s]* %s\n  <%s|%s>",
			emoji, strings.ToUpper(advisory.Severity), summary, advisory.HTMLURL, advisory.ID)
		updates = append(updates, update)
	}
	return updates
}

func (c *SecurityAdvisoryChecker) formatMessage(updates []string) string {
	return fmt.Sprintf("🔐 *GitHub Security Advisories に新しい脆弱性情報があります！* (%d件)\n\n%s",
		len(updates), strings.Join(updates, "\n\n"))
}

// ========================================
// Main Application Logic
// ========================================

// Checker represents an update checker
type Checker interface {
	Check(ctx context.Context) (bool, error)
}

// NamedChecker represents a checker with a name
type NamedChecker struct {
	Name    string
	Checker Checker
}

// App represents the main application
type App struct {
	notifier *SlackNotifier
	checkers []NamedChecker
}

// NewApp creates a new App instance
func NewApp(webhookURL, githubToken string) *App {
	notifier := NewSlackNotifier(webhookURL)

	// Checkers are executed in order
	checkers := []NamedChecker{
		{Name: "GCP Release Notes", Checker: NewGCPReleaseChecker(notifier)},
		{Name: "Go Releases", Checker: NewGoReleaseChecker(notifier, githubToken)},
		{Name: "GitHub Security Advisories", Checker: NewSecurityAdvisoryChecker(notifier, githubToken)},
	}

	return &App{
		notifier: notifier,
		checkers: checkers,
	}
}

// Run executes all update checks
func (a *App) Run(ctx context.Context) error {
	log.Println("Starting update watcher...")

	var errors []string
	hasUpdates := false

	// Run all checkers in order
	for _, nc := range a.checkers {
		updated, err := nc.Checker.Check(ctx)
		if err != nil {
			log.Printf("Error checking %s: %v", nc.Name, err)
			errors = append(errors, fmt.Sprintf("❌ %s: %v", nc.Name, err))
		} else if updated {
			hasUpdates = true
		}
	}

	// Send summary notification
	if err := a.sendSummary(ctx, errors, hasUpdates); err != nil {
		log.Printf("Failed to send summary: %v", err)
		return err
	}

	log.Println("Update watcher completed successfully")
	return nil
}

// sendSummary sends a summary notification based on results
func (a *App) sendSummary(ctx context.Context, errors []string, hasUpdates bool) error {
	now := time.Now().Format("2006-01-02 15:04:05 MST")

	var message string
	switch {
	case len(errors) > 0:
		message = fmt.Sprintf("⚠️ *アップデート監視完了（一部エラーあり）* - %s\n\n%s",
			now, strings.Join(errors, "\n"))
	case !hasUpdates:
		message = fmt.Sprintf("✅ *アップデート監視完了* - %s\n新しい更新はありませんでした。", now)
	default:
		message = fmt.Sprintf("✅ *アップデート監視完了* - %s\n上記の更新を確認してください。", now)
	}

	return a.notifier.Notify(ctx, message)
}

// ========================================
// Main Entry Point
// ========================================

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Validate environment variables
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		log.Fatal("SLACK_WEBHOOK_URL environment variable is not set")
	}

	githubToken := os.Getenv("GITHUB_TOKEN")

	// Create and run application
	ctx := context.Background()
	app := NewApp(webhookURL, githubToken)

	if err := app.Run(ctx); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}
