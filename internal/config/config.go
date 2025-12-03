// Package config provides configuration constants and variables
// for the update-watcher application.
package config

import "time"

// API URLs and endpoints
const (
	GCPReleaseNotesURL          = "https://cloud.google.com/feeds/gcp-release-notes.xml"
	GCPSecurityBulletinsURL     = "https://cloud.google.com/feeds/google-cloud-security-bulletins.xml"
	GoReleasesURL               = "https://api.github.com/repos/golang/go/releases"
	TerraformReleasesURL        = "https://api.github.com/repos/hashicorp/terraform/releases"
	AWSSecurityBulletinsURL     = "https://aws.amazon.com/security/security-bulletins/rss/feed/"
	CloudflareSecurityBlogURL   = "https://blog.cloudflare.com/tag/security/rss/"
	DebianSecurityURL           = "https://www.debian.org/security/dsa-long"
	NVDCVEURL                   = "https://services.nvd.nist.gov/rest/json/cves/2.0"
	GitHubSecurityAdvisoriesURL = "https://api.github.com/advisories"
)

// Behavior configuration
const (
	CheckPeriodHours    = 25 // 過去25時間の更新をチェック（1日1回実行なので余裕を持たせる）
	HTTPTimeout         = 30 * time.Second
	MaxSummaryLength    = 200
	MaxAdvisorySummary  = 150
	MaxDescriptionLines = 3
	GitHubAPIPerPage    = 100
)

// SeverityEmojis maps severity levels to emoji
var SeverityEmojis = map[string]string{
	"critical": "🚨",
	"high":     "❗",
	"medium":   "⚠️",
	"low":      "ℹ️",
}
