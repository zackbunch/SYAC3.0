package gitlab

import (
	"encoding/json"
	"fmt"
)

// BranchesService defines the interface for GitLab branch operations.
type BranchesService interface {
	ListProtectedBranches() ([]ProtectedBranch, error)
}

// branchesService is a concrete implementation of BranchesService.
type branchesService struct {
	client *Client
}

// ListProtectedBranches fetches all protected branches from the project.
func (s *branchesService) ListProtectedBranches() ([]ProtectedBranch, error) {
	path := fmt.Sprintf("/projects/%s/protected_branches", urlEncode(s.client.projectID))
	respData, err := s.client.DoRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch protected branches: %w", err)
	}

	var branches []ProtectedBranch
	if err := json.Unmarshal(respData, &branches); err != nil {
		return nil, fmt.Errorf("failed to parse protected branches list: %w", err)
	}

	return branches, nil
}
