// pkg/gitlab/repo_files.go
package gitlab

import (
	"fmt"
	"net/url"
	"strings"
)

// ---------- Options ----------

type fileBaseOpts struct {
	Branch          string
	CommitMessage   string
	Content         string
	AuthorEmail     string
	AuthorName      string
	Encoding        string // "text" (default) or "base64"
	ExecuteFilemode *bool
	StartBranch     string
}

type CreateFileOptions struct {
	fileBaseOpts
}

type UpdateFileOptions struct {
	fileBaseOpts
	LastCommitID string
}

// ---------- Service ----------

type RepoFilesService interface {
	CreateFile(filePath string, opts CreateFileOptions) error
	UpdateFile(filePath string, opts UpdateFileOptions) error
	UpsertFile(filePath string, branch, commitMessage, content string) error
}

type repoFilesService struct {
	client *Client
}

// ---------- Public methods ----------

func (s *repoFilesService) CreateFile(filePath string, opts CreateFileOptions) error {
	if err := validatePathBranchMsg(filePath, opts.Branch, opts.CommitMessage); err != nil {
		return wrap("CreateFile", err)
	}
	body, err := buildBody(opts.fileBaseOpts, "")
	if err != nil {
		return wrap("CreateFile", err)
	}

	path := fmt.Sprintf("/projects/%s/repository/files/%s",
		urlEncode(s.client.projectID),
		url.PathEscape(filePath),
	)

	if _, err := s.client.DoRequest("POST", path, body); err != nil {
		return fmt.Errorf("CreateFile: POST %s failed: %w", path, err)
	}
	return nil
}

func (s *repoFilesService) UpdateFile(filePath string, opts UpdateFileOptions) error {
	if err := validatePathBranchMsg(filePath, opts.Branch, opts.CommitMessage); err != nil {
		return wrap("UpdateFile", err)
	}
	body, err := buildBody(opts.fileBaseOpts, opts.LastCommitID)
	if err != nil {
		return wrap("UpdateFile", err)
	}

	path := fmt.Sprintf("/projects/%s/repository/files/%s",
		urlEncode(s.client.projectID),
		url.PathEscape(filePath),
	)

	if _, err := s.client.DoRequest("PUT", path, body); err != nil {
		return fmt.Errorf("UpdateFile: PUT %s failed: %w", path, err)
	}
	return nil
}

// UpsertFile creates the file if missing; otherwise updates it.
// Uses status-code detection when available; falls back to substring check.
func (s *repoFilesService) UpsertFile(filePath, branch, commitMessage, content string) error {
	create := CreateFileOptions{
		fileBaseOpts: fileBaseOpts{
			Branch:        branch,
			CommitMessage: commitMessage,
			Content:       content,
			Encoding:      "text",
		},
	}
	if err := s.CreateFile(filePath, create); err == nil {
		return nil
	} else {
		// Prefer typed/status error if your client supports it.
		if isAlreadyExists(err) {
			update := UpdateFileOptions{
				fileBaseOpts: fileBaseOpts{
					Branch:        branch,
					CommitMessage: commitMessage,
					Content:       content,
					Encoding:      "text",
				},
			}
			return s.UpdateFile(filePath, update)
		}
		return err
	}
}

// ---------- Helpers ----------

func validatePathBranchMsg(path, branch, msg string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("filePath is required")
	}
	if strings.TrimSpace(branch) == "" {
		return fmt.Errorf("branch is required")
	}
	if strings.TrimSpace(msg) == "" {
		return fmt.Errorf("commit message is required")
	}
	return nil
}

func normalizeContent(enc, c string) (string, error) {
	enc = strings.ToLower(strings.TrimSpace(enc))
	switch enc {
	case "", "text":
		// GitLab expects text content with a trailing newline; keep idempotent
		return strings.TrimRight(c, "\r\n") + "\n", nil
	case "base64":
		// Accept as-is; caller provides base64 string
		return c, nil
	default:
		return "", fmt.Errorf("unsupported encoding: %q (use \"text\" or \"base64\")", enc)
	}
}

func buildBody(base fileBaseOpts, lastCommitID string) (map[string]any, error) {
	content, err := normalizeContent(base.Encoding, base.Content)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"branch":         base.Branch,
		"commit_message": base.CommitMessage,
		"content":        content,
	}
	if base.AuthorEmail != "" {
		body["author_email"] = base.AuthorEmail
	}
	if base.AuthorName != "" {
		body["author_name"] = base.AuthorName
	}
	if base.Encoding != "" {
		body["encoding"] = strings.ToLower(base.Encoding)
	}
	if base.ExecuteFilemode != nil {
		body["execute_filemode"] = *base.ExecuteFilemode
	}
	if base.StartBranch != "" {
		body["start_branch"] = base.StartBranch
	}
	if lastCommitID != "" {
		body["last_commit_id"] = lastCommitID
	}
	return body, nil
}

func wrap(op string, err error) error {
	return fmt.Errorf("%s: %w", op, err)
}

// isAlreadyExists determines if CreateFile failed due to an existing file.
// If your Client exposes HTTP status codes, check 400/409 with the message:
//
//	{ "message": { "file_path": ["already exists"] } }
//
// Here we fallback to a substring.
func isAlreadyExists(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists")
}
