// Package api provides a simple GitHub REST API client with Bearer token auth and pagination.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/dipsylala/ghas-mcp/credentials"
)

const (
	// apiVersion is the GitHub REST API version header value.
	apiVersion = "2022-11-28"
	// maxPages limits automatic pagination to prevent unbounded calls.
	maxPages = 20
	// perPage is the number of items requested per page.
	perPage = 100
)

// Client is a minimal GitHub REST API client.
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Client using credentials from the environment or gh CLI.
func NewClient() (*Client, error) {
	token, err := credentials.GetToken()
	if err != nil {
		return nil, err
	}
	return &Client{
		token:      token,
		baseURL:    credentials.DefaultBaseURL,
		httpClient: &http.Client{},
	}, nil
}

// GetJSON fetches a single JSON resource from path (relative, e.g. "/repos/owner/repo/...").
// Query parameters are passed as key-value pairs in params.
func (c *Client) GetJSON(ctx context.Context, path string, params url.Values, out interface{}) error {
	reqURL := c.baseURL + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req)

	log.Printf("GET %s", req.URL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseGitHubError(resp.StatusCode, body)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// GetAllPages fetches all pages from a paginated GitHub list endpoint and returns
// the concatenated raw JSON items. Each page must return a JSON array at the top level.
func (c *Client) GetAllPages(ctx context.Context, path string, params url.Values) ([]json.RawMessage, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("per_page", fmt.Sprintf("%d", perPage))

	reqURL := c.baseURL + path + "?" + params.Encode()
	var allItems []json.RawMessage

	for page := 1; reqURL != "" && page <= maxPages; page++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		c.setHeaders(req)

		log.Printf("GET %s (page %d)", req.URL, page)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close() // nolint:errcheck
		if readErr != nil {
			return nil, fmt.Errorf("read response: %w", readErr)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, parseGitHubError(resp.StatusCode, body)
		}

		var pageItems []json.RawMessage
		if err := json.Unmarshal(body, &pageItems); err != nil {
			return nil, fmt.Errorf("decode page %d: %w", page, err)
		}
		allItems = append(allItems, pageItems...)

		reqURL = nextPageURL(resp.Header.Get("Link"))
	}

	return allItems, nil
}

// setHeaders adds the required GitHub API headers to a request.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
}

// githubErrorBody is used to decode GitHub's standard error envelope.
type githubErrorBody struct {
	Message          string `json:"message"`
	DocumentationURL string `json:"documentation_url,omitempty"`
}

// parseGitHubError returns a user-readable error from a non-2xx response.
func parseGitHubError(statusCode int, body []byte) error {
	var gh githubErrorBody
	if json.Unmarshal(body, &gh) == nil && gh.Message != "" {
		return fmt.Errorf("GitHub API error %d: %s", statusCode, gh.Message)
	}
	preview := string(body)
	if len(preview) > 200 {
		preview = preview[:200]
	}
	return fmt.Errorf("GitHub API error %d: %s", statusCode, preview)
}

// nextPageURL parses a GitHub Link header and returns the URL for rel="next", or "".
// Example header: <https://api.github.com/...?page=2>; rel="next", <...>; rel="last"
var reLinkNext = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

func nextPageURL(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}
	for _, part := range strings.Split(linkHeader, ",") {
		if m := reLinkNext.FindStringSubmatch(strings.TrimSpace(part)); m != nil {
			return m[1]
		}
	}
	return ""
}
