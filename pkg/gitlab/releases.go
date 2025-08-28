package gitlab

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ReleasesService defines the interface for GitLab Release operations.
type ReleasesService interface {
	CreateRelease(payload ReleasePayload) error
	GetLatestRelease() (Release, error)
}

// releasesService is a concrete implementation of ReleasesService.
type releasesService struct {
	client *Client
}

var ErrNoReleases = errors.New("no releases")

// CreateRelease creates a new release in the project.
func (s *releasesService) CreateRelease(payload ReleasePayload) error {
	path := fmt.Sprintf("/projects/%s/releases", urlEncode(s.client.projectID))

	_, err := s.client.DoRequest("POST", path, payload)
	if err != nil {
		return fmt.Errorf("failed to create release %q: %w", payload.TagName, err)
	}

	fmt.Printf("✅ Created GitLab release: %s (%s)\n", payload.Name, payload.TagName)
	return nil
}

// GetLatestRelease returns the most-recent release (by created_at).
// Safe against nil receiver/client and projects with zero releases.
func (s *releasesService) GetLatestRelease() (Release, error) {
	if s == nil || s.client == nil {
		return Release{}, fmt.Errorf("GetLatestRelease: nil client")
	}
	pid := strings.TrimSpace(s.client.projectID)
	if pid == "" {
		return Release{}, fmt.Errorf("GetLatestRelease: empty projectID")
	}

	path := fmt.Sprintf(
		"/projects/%s/releases?per_page=1&order_by=created_at&sort=desc",
		urlEncode(pid),
	)

	respData, err := s.client.DoRequest("GET", path, nil)
	if err != nil {
		// Normalize "no releases yet" to a sentinel instead of surfacing a 404
		if gerr, ok := err.(*GitLabError); ok && gerr.StatusCode == 404 {
			return Release{}, ErrNoReleases
		}
		return Release{}, fmt.Errorf("GetLatestRelease: fetch failed: %w", err)
	}

	var releases []Release
	if err := json.Unmarshal(respData, &releases); err != nil {
		return Release{}, fmt.Errorf("GetLatestRelease: unmarshal failed: %w", err)
	}
	if len(releases) == 0 {
		return Release{}, ErrNoReleases
	}
	return releases[0], nil
}
