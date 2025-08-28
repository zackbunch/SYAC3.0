package gitlab

import (
	"syac/internal/version"
)

// ---------- Public types / interface ----------

type MergeRequestsService interface {
	GetMergeRequestDescription(mrID string) (string, error)
	UpdateMergeRequestDescription(mrID string, newDescription string) error
	InsertReleaseTypeInDescription(mrID string) error

	CreateMergeRequestComment(mrID string) error
	UpsertMergeRequestComment(mrID string) error
	

	GetVersionBump(mrID string) (version.VersionType, error)
	GetMergeRequestForCommit(sha string) (MergeRequest, error)
	GetLatestMergeRequest() (MergeRequest, error)

	ListNotes(projectID, mrID string) ([]Note, error)
	UpdateNote(projectID, mrID string, noteID int, body string) error
	CreateNote(projectID, mrID string, body string) error
}

type mrsService struct {
	client *Client
}
