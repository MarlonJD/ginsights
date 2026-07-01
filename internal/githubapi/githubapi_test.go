package githubapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTokenFromEnvPrefersGinsightsToken(t *testing.T) {
	token, source := TokenFromEnv(func(key string) string {
		switch key {
		case "GINSIGHTS_GITHUB_TOKEN":
			return "preferred"
		case "GITHUB_TOKEN":
			return "fallback"
		default:
			return ""
		}
	})
	if token != "preferred" || source != "GINSIGHTS_GITHUB_TOKEN" {
		t.Fatalf("TokenFromEnv = %q, %q; want preferred GINSIGHTS_GITHUB_TOKEN", token, source)
	}
}

func TestRedactTokenRemovesSecret(t *testing.T) {
	got := RedactToken("request failed with token secret-token", "secret-token")
	if strings.Contains(got, "secret-token") {
		t.Fatalf("redacted message still contains token: %q", got)
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("redacted message = %q, want redaction marker", got)
	}
}

func TestClientFetchesRepositoryAndTrafficMetrics(t *testing.T) {
	var authHeaders []string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		var body string
		switch r.URL.Path {
		case "/repos/acme/widgets":
			body = `{"full_name":"acme/widgets","stargazers_count":42,"forks_count":7,"open_issues_count":3}`
		case "/repos/acme/widgets/traffic/views":
			body = `{"count":120,"uniques":45}`
		case "/repos/acme/widgets/traffic/clones":
			body = `{"count":12,"uniques":8}`
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		return jsonResponse(http.StatusOK, body), nil
	})

	client := Client{BaseURL: "https://api.github.test", Token: "secret-token", HTTPClient: &http.Client{Transport: transport}}
	metrics, err := client.Fetch(context.Background(), "acme/widgets")
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Repository != "acme/widgets" || metrics.Stars != 42 || metrics.Forks != 7 || metrics.OpenIssues != 3 {
		t.Fatalf("repository metrics = %+v, want repo metadata", metrics)
	}
	if metrics.Views == nil || metrics.Views.Count != 120 || metrics.Views.Uniques != 45 {
		t.Fatalf("views = %+v, want traffic views", metrics.Views)
	}
	if metrics.Clones == nil || metrics.Clones.Count != 12 || metrics.Clones.Uniques != 8 {
		t.Fatalf("clones = %+v, want traffic clones", metrics.Clones)
	}
	for _, got := range authHeaders {
		if got != "Bearer secret-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
	}
}

func TestClientErrorsDoNotExposeToken(t *testing.T) {
	client := Client{
		BaseURL: "https://api.github.test",
		Token:   "secret-token",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusUnauthorized, "nope"), nil
		})},
	}
	_, err := client.Fetch(context.Background(), "acme/widgets")
	if err == nil {
		t.Fatal("Fetch succeeded, want error")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error contains token: %v", err)
	}
}

func TestClientRejectsInvalidRepositoryBeforeRequest(t *testing.T) {
	called := false
	client := Client{
		BaseURL: "https://api.github.test",
		Token:   "secret-token",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			called = true
			return jsonResponse(http.StatusOK, "{}"), nil
		})},
	}
	_, err := client.Fetch(context.Background(), "acme/widgets?token=secret-token")
	if err == nil {
		t.Fatal("Fetch succeeded, want invalid repository error")
	}
	if called {
		t.Fatal("HTTP client was called for invalid repository")
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("error contains token-like query: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
