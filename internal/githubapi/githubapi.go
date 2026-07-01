package githubapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.github.com"

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type Metrics struct {
	Repository string         `json:"repository"`
	Stars      int            `json:"stars"`
	Forks      int            `json:"forks"`
	OpenIssues int            `json:"open_issues"`
	Views      *TrafficMetric `json:"views,omitempty"`
	Clones     *TrafficMetric `json:"clones,omitempty"`
	Warnings   []string       `json:"warnings,omitempty"`
}

type TrafficMetric struct {
	Count   int `json:"count"`
	Uniques int `json:"uniques"`
}

type repositoryResponse struct {
	FullName        string `json:"full_name"`
	StargazersCount int    `json:"stargazers_count"`
	ForksCount      int    `json:"forks_count"`
	OpenIssuesCount int    `json:"open_issues_count"`
}

func TokenFromEnv(lookup func(string) string) (string, string) {
	for _, key := range []string{"GINSIGHTS_GITHUB_TOKEN", "GITHUB_TOKEN"} {
		if token := strings.TrimSpace(lookup(key)); token != "" {
			return token, key
		}
	}
	return "", ""
}

func RedactToken(message, token string) string {
	if token == "" {
		return message
	}
	return strings.ReplaceAll(message, token, "[redacted]")
}

func (c Client) Fetch(ctx context.Context, repo string) (Metrics, error) {
	repo = strings.TrimSpace(repo)
	if !validRepo(repo) {
		return Metrics{}, fmt.Errorf("invalid GitHub repository: use owner/name")
	}
	var repoData repositoryResponse
	if err := c.getJSON(ctx, "/repos/"+repo, &repoData); err != nil {
		return Metrics{}, err
	}
	if repoData.FullName == "" {
		repoData.FullName = repo
	}
	metrics := Metrics{
		Repository: repoData.FullName,
		Stars:      repoData.StargazersCount,
		Forks:      repoData.ForksCount,
		OpenIssues: repoData.OpenIssuesCount,
	}

	var views TrafficMetric
	if err := c.getJSON(ctx, "/repos/"+repo+"/traffic/views", &views); err != nil {
		metrics.Warnings = append(metrics.Warnings, "views unavailable: "+err.Error())
	} else {
		metrics.Views = &views
	}
	var clones TrafficMetric
	if err := c.getJSON(ctx, "/repos/"+repo+"/traffic/clones", &clones); err != nil {
		metrics.Warnings = append(metrics.Warnings, "clones unavailable: "+err.Error())
	} else {
		metrics.Clones = &clones
	}
	return metrics, nil
}

func (c Client) getJSON(ctx context.Context, path string, dst any) error {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create github request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ginsights")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("github api request failed: %s", RedactToken(err.Error(), c.Token))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return fmt.Errorf("github api %s failed: %s", resp.Status, RedactToken(message, c.Token))
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode github api response: %w", err)
	}
	return nil
}

func validRepo(repo string) bool {
	owner, name, ok := strings.Cut(repo, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return false
	}
	return !strings.ContainsAny(owner, " ?#") && !strings.ContainsAny(name, " ?#")
}

func DefaultClient(token string) Client {
	return Client{BaseURL: defaultBaseURL, Token: token}
}

func EnvToken() (string, string) {
	return TokenFromEnv(os.Getenv)
}
