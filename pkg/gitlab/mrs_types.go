package gitlab

import "errors"

var (
	syacMarker         = "<!-- syac:release-type -->"
	ErrNoMergeRequests = errors.New("no merge requests")
)

type Note struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}
