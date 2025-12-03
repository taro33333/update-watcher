package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"update-watcher/internal/checker"
	"update-watcher/internal/notifier"
	"update-watcher/internal/sources"
)

// App represents the main application
type App struct {
	notifier         *notifier.SlackNotifier
	securityNotifier *notifier.SlackNotifier
	checkers         []checker.Named
}

// NewApp creates a new App instance
func NewApp(webhookURL, securityWebhookURL, githubToken string) *App {
	n := notifier.New(webhookURL)

	// Use dedicated security webhook if provided, otherwise use the same notifier
	securityN := n
	if securityWebhookURL != "" && securityWebhookURL != webhookURL {
		securityN = notifier.New(securityWebhookURL)
		log.Println("Using dedicated security notification channel")
	}

	// Checkers are executed in order
	checkers := []checker.Named{
		{Name: "GCP Release Notes", Checker: sources.NewGCP(n)},
		{Name: "Go Releases", Checker: sources.NewGo(n, githubToken)},
		{Name: "Terraform Releases", Checker: sources.NewTerraform(n, githubToken)},
		{Name: "AWS Security Bulletins", Checker: sources.NewAWS(securityN)},
		{Name: "Cloudflare Security Blog", Checker: sources.NewCloudflare(securityN)},
		{Name: "GCP Security Bulletins", Checker: sources.NewGCPSecurity(securityN)},
		{Name: "Debian Security Advisories", Checker: sources.NewDebian(securityN)},
		{Name: "NVD CVE Database", Checker: sources.NewNVD(securityN)},
		{Name: "GitHub Security Advisories", Checker: sources.NewGitHub(securityN, githubToken)},
	}

	return &App{
		notifier:         n,
		securityNotifier: securityN,
		checkers:         checkers,
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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Validate environment variables
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		log.Fatal("SLACK_WEBHOOK_URL environment variable is not set")
	}

	// Optional: dedicated webhook for security notifications
	securityWebhookURL := os.Getenv("SLACK_SECURITY_WEBHOOK_URL")
	githubToken := os.Getenv("GITHUB_TOKEN")

	// Create and run application
	ctx := context.Background()
	app := NewApp(webhookURL, securityWebhookURL, githubToken)

	if err := app.Run(ctx); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}
