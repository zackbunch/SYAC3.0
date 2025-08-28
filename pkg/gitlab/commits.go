package gitlab

import (
	"encoding/json"
	"fmt"
)

// CommitsService defines the interface for GitLab Commit operations.
type CommitsService interface {
	GetCommit(sha string) (Commit, error)
}

// commitsService is a concrete implementation of CommitsService.
type commitsService struct {
	client *Client // A reference to the base GitLab client
}

// Commit represents a GitLab Commit object.
type Commit struct {
	ID             string   `json:"id"`
	ShortID        string   `json:"short_id"`
	Title          string   `json:"title"`
	Message        string   `json:"message"`
	ParentIDs      []string `json:"parent_ids"`
	CommitterName  string   `json:"committer_name"`
	CommitterEmail string   `json:"committer_email"`
	CommittedDate  string   `json:"committed_date"`
}

// GetCommit fetches a single commit from the project.
func (s *commitsService) GetCommit(sha string) (Commit, error) {
	path := fmt.Sprintf("/projects/%s/repository/commits/%s", urlEncode(s.client.projectID), sha)
	respData, err := s.client.DoRequest("GET", path, nil)
	if err != nil {
		return Commit{}, fmt.Errorf("failed to get commit %s: %w", sha, err)
	}

	var commit Commit
	if err := json.Unmarshal(respData, &commit); err != nil {
		return Commit{}, fmt.Errorf("failed to unmarshal commit data: %w", err)
	}

	return commit, nil
}
