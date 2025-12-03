// Package client provides HTTP client utilities for fetching data
// from various APIs and endpoints.
package client

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"update-watcher/internal/config"
)

// HTTPClient is a shared HTTP client for all requests
var HTTPClient = &http.Client{
	Timeout: config.HTTPTimeout,
}

// FetchURL performs a simple HTTP GET request
func FetchURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := HTTPClient.Do(req)
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

// FetchGitHubAPI performs a GitHub API request with authentication
func FetchGitHubAPI(ctx context.Context, url, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := HTTPClient.Do(req)
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
