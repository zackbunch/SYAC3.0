package gitlab

import (
	"encoding/json"
	"fmt"
	"strings"

	"syac/internal/assets"
)

// ---------- Notes (comments) ----------

func (s *mrsService) CreateMergeRequestComment(mrID string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("CreateMergeRequestComment: nil client")
	}
	contentBytes, err := assets.MrCommentContent.ReadFile("mr_comment.md")
	if err != nil {
		return fmt.Errorf("CreateMergeRequestComment: read embedded: %w", err)
	}
	return s.CreateNote(s.client.projectID, mrID, string(contentBytes))
}

func (s *mrsService) UpsertMergeRequestComment(mrID string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("UpsertMergeRequestComment: nil client")
	}
	contentBytes, err := assets.MrCommentContent.ReadFile("mr_comment.md")
	if err != nil {
		return fmt.Errorf("UpsertMergeRequestComment: read embedded: %w", err)
	}
	body := string(contentBytes)

	notes, err := s.ListNotes(s.client.projectID, mrID)
	if err != nil {
		return fmt.Errorf("UpsertMergeRequestComment: list notes: %w", err)
	}

	var existingID int
	for _, n := range notes {
		if strings.Contains(n.Body, syacMarker) {
			existingID = n.ID
			break
		}
	}

	if existingID == 0 {
		return s.CreateNote(s.client.projectID, mrID, body)
	}
	return s.UpdateNote(s.client.projectID, mrID, existingID, body)
}

func (s *mrsService) ListNotes(projectID, mrID string) ([]Note, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("ListNotes: nil client")
	}
	path := fmt.Sprintf("/projects/%s/merge_requests/%s/notes", urlEncode(projectID), mrID)
	data, err := s.client.DoRequest("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("ListNotes: %w", err)
	}
	var notes []Note
	if err := json.Unmarshal(data, &notes); err != nil {
		return nil, fmt.Errorf("ListNotes: unmarshal: %w", err)
	}
	return notes, nil
}

func (s *mrsService) CreateNote(projectID, mrID string, body string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("CreateNote: nil client")
	}
	path := fmt.Sprintf("/projects/%s/merge_requests/%s/notes", urlEncode(projectID), mrID)
	_, err := s.client.DoRequest("POST", path, map[string]string{"body": body})
	if err != nil {
		return fmt.Errorf("CreateNote: %w", err)
	}
	return nil
}

func (s *mrsService) UpdateNote(projectID, mrID string, noteID int, body string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("UpdateNote: nil client")
	}
	path := fmt.Sprintf("/projects/%s/merge_requests/%s/notes/%d", urlEncode(projectID), mrID, noteID)
	_, err := s.client.DoRequest("PUT", path, map[string]string{"body": body})
	if err != nil {
		return fmt.Errorf("UpdateNote: %w", err)
	}
	return nil
}
