package gitlab

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	projectID  string

	// Services
	MergeRequests MergeRequestsService
	Tags          TagsService
	Commits       CommitsService
	Releases      ReleasesService // New service
	Branches      BranchesService
	Repositories  RepoFilesService
}

// GitLabError represents an error response from the GitLab API.
type GitLabError struct {
	StatusCode int
	Message    string
	Body       []byte
}

// Error returns a string representation of the GitLabError.
func (e *GitLabError) Error() string {
	return fmt.Sprintf("GitLab API error (%d): %s -- %s", e.StatusCode, e.Message, string(e.Body))
}

// NewClient creates a new GitLab client using environment variables for configuration.
// It expects a Personal Access Token or Project Access Token.
// Required environment variables:
//   - SYAC_GITLAB_API_TOKEN (preferred) or GITLAB_API_TOKEN
//   - CI_API_V4_URL (if running in CI) or GITLAB_BASE_URL (if running locally)
//   - CI_PROJECT_ID (if running in CI) or GITLAB_PROJECT_ID (if running locally)
//
// An optional GITLAB_CLIENT_TIMEOUT_SECONDS can be set to configure the HTTP client timeout.
func NewClient() (*Client, error) {
	var baseURL, token, projectID string

	isPipeline := os.Getenv("GITLAB_CI") == "true"

	// Prioritize SYAC_GITLAB_API_TOKEN
	token = os.Getenv("SYAC_GITLAB_API_TOKEN")
	if token == "" {
		// Fallback to GITLAB_API_TOKEN
		token = os.Getenv("GITLAB_API_TOKEN")
	}

	if isPipeline {
		baseURL = strings.TrimSuffix(os.Getenv("CI_API_V4_URL"), "/api/v4")
		projectID = os.Getenv("CI_PROJECT_ID")
	} else {
		baseURL = os.Getenv("GITLAB_BASE_URL")
		projectID = os.Getenv("GITLAB_PROJECT_ID")
	}

	if token == "" {
		return nil, errors.New("SYAC_GITLAB_API_TOKEN or GITLAB_API_TOKEN must be set")
	}

	if projectID == "" {
		return nil, errors.New("CI_PROJECT_ID or GITLAB_PROJECT_ID must be set")
	}

	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, errors.New("invalid GitLab base URL: " + err.Error())
	}

	// Optional timeout override
	timeout := 10 * time.Second
	if timeoutStr := os.Getenv("GITLAB_CLIENT_TIMEOUT_SECONDS"); timeoutStr != "" {
		if seconds, err := strconv.Atoi(timeoutStr); err == nil && seconds > 0 {
			timeout = time.Duration(seconds) * time.Second
		}
	}

	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		projectID: projectID,
	}

	// Initialize services
	c.MergeRequests = &mrsService{client: c}
	c.Tags = &tagsService{client: c}
	c.Commits = &commitsService{client: c}
	c.Releases = &releasesService{client: c}
	c.Branches = &branchesService{client: c}
	c.Repositories = &repoFilesService{client: c}

	return c, nil
}

// DoRequest sends an HTTP request to the GitLab API and returns the response body.
// It handles request creation, authentication, execution, and error parsing.
// The 'path' should be relative to the /api/v4 endpoint (e.g., "/projects/123/merge_requests/456").
func (c *Client) DoRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	fullURL := fmt.Sprintf("%s/api/v4%s", c.baseURL, path)
	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request [%s %s]: %w", method, fullURL, err)
	}

	req.Header.Set("PRIVATE-TOKEN", c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed [%s %s]: %w", method, fullURL, err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &GitLabError{
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
			Body:       respData,
		}
	}

	return respData, nil
}

// urlEncode safely encodes a GitLab project path (e.g., "group/project" -> "group%2Fproject").
func urlEncode(s string) string {
	return url.PathEscape(s)
}
