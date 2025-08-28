package gitlab

import (
	"encoding/json"
	"fmt"
	"net/url"
)

func (s *mrsService) GetMergeRequestForCommit(sha string) (MergeRequest, error) {
	if s == nil || s.client == nil {
		return MergeRequest{}, fmt.Errorf("GetMergeRequestForCommit: nil client")
	}
	path := fmt.Sprintf("/projects/%s/repository/commits/%s/merge_requests", urlEncode(s.client.projectID), sha)
	respData, err := s.client.DoRequest("GET", path, nil)
	if err != nil {
		return MergeRequest{}, fmt.Errorf("GetMergeRequestForCommit: %w", err)
	}
	var mrs []MergeRequest
	if err := json.Unmarshal(respData, &mrs); err != nil {
		return MergeRequest{}, fmt.Errorf("GetMergeRequestForCommit: unmarshal: %w", err)
	}
	if len(mrs) == 0 {
		return MergeRequest{}, fmt.Errorf("GetMergeRequestForCommit: none for commit %s", sha)
	}
	return mrs[0], nil
}

// GetLatestMergeRequest: most-recent by updated_at (more useful for automation).
// Returns ErrNoMergeRequests if none found so callers can branch cleanly.
func (s *mrsService) GetLatestMergeRequest() (MergeRequest, error) {
	if s == nil || s.client == nil {
		return MergeRequest{}, fmt.Errorf("GetLatestMergeRequest: nil client")
	}

	q := url.Values{}
	q.Set("per_page", "1")
	q.Set("order_by", "updated_at") // better than created_at for "latest"
	q.Set("sort", "desc")

	path := fmt.Sprintf("/projects/%s/merge_requests?%s", urlEncode(s.client.projectID), q.Encode())
	respData, err := s.client.DoRequest("GET", path, nil)
	if err != nil {
		return MergeRequest{}, fmt.Errorf("GetLatestMergeRequest: fetch failed: %w", err)
	}

	var mrs []MergeRequest
	if err := json.Unmarshal(respData, &mrs); err != nil {
		return MergeRequest{}, fmt.Errorf("GetLatestMergeRequest: unmarshal failed: %w", err)
	}
	if len(mrs) == 0 {
		return MergeRequest{}, ErrNoMergeRequests
	}
	return mrs[0], nil
}
