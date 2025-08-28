package gitlab

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"syac/internal/version"
)

type TagsService interface {
	ListProjectTags() ([]Tag, error)
	GetLatestTag() (version.Version, error)
	CreateTag(tagName, ref, message string) error
	GetNextVersion(bump version.VersionType) (version.Version, version.Version, error)
}

type tagsService struct {
	client *Client
}

// ListProjectTags retrieves all tags in the current project.
// If the project has no tags, it returns an empty slice.
func (s *tagsService) ListProjectTags() ([]Tag, error) {
	path := fmt.Sprintf("/projects/%s/repository/tags", urlEncode(s.client.projectID))
	respData, err := s.client.DoRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tags: %w", err)
	}

	var tags []Tag
	if err := json.Unmarshal(respData, &tags); err != nil {
		return nil, fmt.Errorf("failed to parse tag list: %w", err)
	}
	return tags, nil
}

// GetLatestTag finds the highest SemVer-compliant tag from the repo.
// Rejects tags with "v" prefix (e.g. "v1.2.3"). If no valid tags exist,
// it defaults to version 0.0.0.
func (s *tagsService) GetLatestTag() (version.Version, error) {
	tags, err := s.ListProjectTags()
	if err != nil {
		return version.Version{Major: 0, Minor: 0, Patch: 0}, nil
	}

	var parsed []version.Version
	for _, tag := range tags {
		v, perr := version.Parse(tag.Name)
		if perr != nil {
			// Ignore non-SemVer (including "v1.2.3")
			continue
		}
		parsed = append(parsed, v)
	}

	if len(parsed) == 0 {
		return version.Version{Major: 0, Minor: 0, Patch: 0}, nil
	}

	// Sort ascending and return the highest
	sort.Slice(parsed, func(i, j int) bool {
		return parsed[i].LessThan(parsed[j])
	})
	return parsed[len(parsed)-1], nil
}

// CreateTag creates a new Git tag for the given ref and optional message.
// Also sets SYAC_TAG in the environment for downstream jobs.
func (s *tagsService) CreateTag(tagName, ref, message string) error {
	path := fmt.Sprintf("/projects/%s/repository/tags", urlEncode(s.client.projectID))

	payload := map[string]string{
		"tag_name": tagName,
		"ref":      ref,
	}
	if message != "" {
		payload["message"] = message
	}

	if _, err := s.client.DoRequest("POST", path, payload); err != nil {
		return fmt.Errorf("failed to create tag %q on ref %q: %w", tagName, ref, err)
	}

	if err := os.Setenv("SYAC_TAG", tagName); err != nil {
		return fmt.Errorf("failed to set SYAC_TAG: %w", err)
	}
	return nil
}

// GetNextVersion calculates the next version by bump type.
// If no tags exist, it starts from 0.0.0.
func (s *tagsService) GetNextVersion(bump version.VersionType) (version.Version, version.Version, error) {
	current, _ := s.GetLatestTag()
	next := current.Increment(bump)
	return current, next, nil
}
