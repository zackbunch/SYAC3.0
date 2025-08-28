package gitlab

import (
	"fmt"
	"regexp"
	"strings"

	"syac/internal/version"
)

func ParseVersionBump(text string) (version.VersionType, bool) {
	checkboxRe := regexp.MustCompile(`- \[x\] \*\*(Patch|Minor|Major)\*\*`)
	for _, line := range strings.Split(text, "\n") {
		if m := checkboxRe.FindStringSubmatch(line); len(m) > 1 {
			return version.VersionType(m[1]), true
		}
	}
	return "", false
}

func (s *mrsService) GetVersionBump(mrID string) (version.VersionType, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("GetVersionBump: nil client")
	}

	// 1) Prefer the SYAC note. If there are multiple, prefer the most recent.
	if notes, err := s.ListNotes(s.client.projectID, mrID); err == nil && len(notes) > 0 {
		// iterate from newest to oldest (GitLab often returns ascending; play it safe)
		for i := len(notes) - 1; i >= 0; i-- {
			n := notes[i]
			if strings.Contains(n.Body, syacMarker) {
				if bump, ok := ParseVersionBump(n.Body); ok {
					return bump, nil
				}
				// Found our note but no box checked; keep looking for an older one
				// If none match, we’ll fall back to description.
			}
		}
	}

	// 2) Fallback: parse MR description
	desc, err := s.GetMergeRequestDescription(mrID)
	if err != nil {
		return "", fmt.Errorf("GetVersionBump: %w", err)
	}
	if bump, ok := ParseVersionBump(desc); ok {
		return bump, nil
	}

	// 3) Nothing selected anywhere → default
	return version.Patch, nil
}
