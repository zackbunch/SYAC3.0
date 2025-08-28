package gitlab

import (
	"encoding/json"
	"fmt"
	"strings"

	"syac/internal/assets"
)

func (s *mrsService) GetMergeRequestDescription(mrID string) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("GetMergeRequestDescription: nil client")
	}
	path := fmt.Sprintf("/projects/%s/merge_requests/%s", urlEncode(s.client.projectID), mrID)
	respData, err := s.client.DoRequest("GET", path, nil)
	if err != nil {
		return "", fmt.Errorf("GetMergeRequestDescription: %w", err)
	}
	var mr struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(respData, &mr); err != nil {
		return "", fmt.Errorf("GetMergeRequestDescription: unmarshal: %w", err)
	}
	return mr.Description, nil
}

func (s *mrsService) UpdateMergeRequestDescription(mrID string, newDescription string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("UpdateMergeRequestDescription: nil client")
	}
	path := fmt.Sprintf("/projects/%s/merge_requests/%s", urlEncode(s.client.projectID), mrID)
	payload := map[string]string{"description": newDescription}
	if _, err := s.client.DoRequest("PUT", path, payload); err != nil {
		return fmt.Errorf("UpdateMergeRequestDescription: %w", err)
	}
	return nil
}

// InsertReleaseTypeInDescription ensures the SYAC release-type block is present
// in the MR description. If already present, it leaves the description untouched.
func (s *mrsService) InsertReleaseTypeInDescription(mrID string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("InsertReleaseTypeInDescription: nil client")
	}

	// Load the embedded SYAC block (same block used for MR comment).
	contentBytes, err := assets.MrCommentContent.ReadFile("mr_comment.md")
	if err != nil {
		return fmt.Errorf("InsertReleaseTypeInDescription: read embedded: %w", err)
	}
	block := string(contentBytes)

	// Get current description.
	desc, err := s.GetMergeRequestDescription(mrID)
	if err != nil {
		return fmt.Errorf("InsertReleaseTypeInDescription: get description: %w", err)
	}

	// If marker already present, nothing to do.
	if strings.Contains(desc, syacMarker) {
		return nil
	}

	// Append the block with spacing.
	var newDesc string
	if strings.TrimSpace(desc) == "" {
		newDesc = block + "\n"
	} else {
		newDesc = strings.TrimRight(desc, "\r\n") + "\n\n" + block + "\n"
	}

	// Push update back to GitLab.
	if err := s.UpdateMergeRequestDescription(mrID, newDesc); err != nil {
		return fmt.Errorf("InsertReleaseTypeInDescription: update: %w", err)
	}

	return nil
}
