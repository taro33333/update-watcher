// Package sources provides update checkers for various information sources
// including GCP, Go, Terraform, Debian, and GitHub.
package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"update-watcher/internal/checker"
	"update-watcher/internal/client"
	"update-watcher/internal/config"
	"update-watcher/internal/notifier"
	"update-watcher/internal/util"
)

// NVD checks for NVD CVE updates
type NVD struct {
	notifier *notifier.SlackNotifier
}

// NewNVD creates a new NVD checker
func NewNVD(n *notifier.SlackNotifier) *NVD {
	return &NVD{notifier: n}
}

// fetchNVDAPI performs a request to NVD API with proper headers
func (c *NVD) fetchNVDAPI(ctx context.Context, apiURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// NVD API requires User-Agent
	req.Header.Set("User-Agent", "update-watcher/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}

// Check implements the Checker interface
func (c *NVD) Check(ctx context.Context) (bool, error) {
	log.Println("Checking NVD CVE Database...")

	// Note: NVD API v2.0 lastModStartDate parameter has issues
	// For now, fetch recent CVEs and filter client-side
	params := url.Values{}
	params.Add("resultsPerPage", "100")
	apiURL := fmt.Sprintf("%s?%s", config.NVDCVEURL, params.Encode())

	// NVD API requires proper User-Agent header
	data, err := c.fetchNVDAPI(ctx, apiURL)
	if err != nil {
		return false, fmt.Errorf("failed to fetch NVD CVE data: %w", err)
	}

	var response checker.NVDResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return false, fmt.Errorf("failed to parse NVD response: %w", err)
	}

	recentCVEs := c.filterRecentCVEs(response.Vulnerabilities)
	if len(recentCVEs) == 0 {
		log.Println("No recent NVD CVEs found")
		return false, nil
	}

	message := c.formatMessage(recentCVEs, response.TotalResults)
	if err := c.notifier.Notify(ctx, message); err != nil {
		return false, err
	}

	return true, nil
}

func (c *NVD) filterRecentCVEs(vulnerabilities []checker.NVDVulnerability) []string {
	var updates []string
	for _, vuln := range vulnerabilities {
		cve := vuln.CVE

		// Check if recently modified
		if !util.IsRecent(cve.LastModified, config.CheckPeriodHours) {
			continue
		}

		// Get English description
		description := c.getEnglishDescription(cve.Descriptions)
		description = util.TruncateText(description, config.MaxAdvisorySummary)

		// Get severity
		severity := c.getSeverity(cve.Metrics)
		severityEmoji := c.getSeverityEmoji(severity)

		// Get CVSS score
		score := c.getCVSSScore(cve.Metrics)
		scoreStr := ""
		if score > 0 {
			scoreStr = fmt.Sprintf(" (CVSS: %.1f)", score)
		}

		// Format update message
		update := fmt.Sprintf("%s *%s* [%s]%s\n  %s\n  <https://nvd.nist.gov/vuln/detail/%s|詳細を見る>",
			severityEmoji, cve.ID, severity, scoreStr, description, cve.ID)
		updates = append(updates, update)
	}
	return updates
}

func (c *NVD) getEnglishDescription(descriptions []checker.NVDDescription) string {
	for _, desc := range descriptions {
		if desc.Lang == "en" {
			return desc.Value
		}
	}
	if len(descriptions) > 0 {
		return descriptions[0].Value
	}
	return "No description available"
}

func (c *NVD) getSeverity(metrics checker.NVDMetrics) string {
	// Try CVSS v3.1 first (most recent)
	if len(metrics.CVSSMetricV31) > 0 {
		for _, metric := range metrics.CVSSMetricV31 {
			if metric.Type == "Primary" && metric.BaseSeverity != "" {
				return metric.BaseSeverity
			}
		}
		if metrics.CVSSMetricV31[0].BaseSeverity != "" {
			return metrics.CVSSMetricV31[0].BaseSeverity
		}
	}

	// Try CVSS v3.0
	if len(metrics.CVSSMetricV30) > 0 {
		for _, metric := range metrics.CVSSMetricV30 {
			if metric.Type == "Primary" && metric.BaseSeverity != "" {
				return metric.BaseSeverity
			}
		}
		if metrics.CVSSMetricV30[0].BaseSeverity != "" {
			return metrics.CVSSMetricV30[0].BaseSeverity
		}
	}

	// Try CVSS v2
	if len(metrics.CVSSMetricV2) > 0 {
		for _, metric := range metrics.CVSSMetricV2 {
			if metric.Type == "Primary" && metric.BaseSeverity != "" {
				return metric.BaseSeverity
			}
		}
		if metrics.CVSSMetricV2[0].BaseSeverity != "" {
			return metrics.CVSSMetricV2[0].BaseSeverity
		}
	}

	return "UNKNOWN"
}

func (c *NVD) getCVSSScore(metrics checker.NVDMetrics) float64 {
	// Try CVSS v3.1 first
	if len(metrics.CVSSMetricV31) > 0 {
		for _, metric := range metrics.CVSSMetricV31 {
			if metric.Type == "Primary" && metric.CVSSData.BaseScore > 0 {
				return metric.CVSSData.BaseScore
			}
		}
		if metrics.CVSSMetricV31[0].CVSSData.BaseScore > 0 {
			return metrics.CVSSMetricV31[0].CVSSData.BaseScore
		}
	}

	// Try CVSS v3.0
	if len(metrics.CVSSMetricV30) > 0 {
		for _, metric := range metrics.CVSSMetricV30 {
			if metric.Type == "Primary" && metric.CVSSData.BaseScore > 0 {
				return metric.CVSSData.BaseScore
			}
		}
		if metrics.CVSSMetricV30[0].CVSSData.BaseScore > 0 {
			return metrics.CVSSMetricV30[0].CVSSData.BaseScore
		}
	}

	// Try CVSS v2
	if len(metrics.CVSSMetricV2) > 0 {
		for _, metric := range metrics.CVSSMetricV2 {
			if metric.Type == "Primary" && metric.CVSSData.BaseScore > 0 {
				return metric.CVSSData.BaseScore
			}
		}
		if metrics.CVSSMetricV2[0].CVSSData.BaseScore > 0 {
			return metrics.CVSSMetricV2[0].CVSSData.BaseScore
		}
	}

	return 0
}

func (c *NVD) getSeverityEmoji(severity string) string {
	severityEmojis := map[string]string{
		"CRITICAL": "🚨",
		"HIGH":     "❗",
		"MEDIUM":   "⚠️",
		"LOW":      "ℹ️",
		"UNKNOWN":  "❓",
	}

	if emoji, ok := severityEmojis[strings.ToUpper(severity)]; ok {
		return emoji
	}
	return "⚠️"
}

func (c *NVD) formatMessage(updates []string, totalResults int) string {
	header := fmt.Sprintf("🛡️ *NVD CVE Database に新しい脆弱性情報があります！* (%d件)",
		len(updates))

	if totalResults > len(updates) {
		header = fmt.Sprintf("🛡️ *NVD CVE Database に新しい脆弱性情報があります！* (%d件 / 全%d件中)",
			len(updates), totalResults)
	}

	return fmt.Sprintf("%s\n\n%s", header, strings.Join(updates, "\n\n"))
}
