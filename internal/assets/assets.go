package assets

import (
	"embed"
	"fmt"
)

//go:embed mr_comment.md
var MrCommentContent embed.FS

// MRCommentTemplate loads the embedded mr_comment.md as a string.
func MRCommentTemplate() string {
	data, err := MrCommentContent.ReadFile("mr_comment.md")
	if err != nil {
		// fail-safe: return a marker so we don't post blank
		return fmt.Sprintf("<!-- syac:release-type --> (error reading mr_comment.md: %v)", err)
	}
	return string(data)
}
