package runtime

import (
	"os"
	"strings"
)

// --- helpers ---
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
func getEnvOrNone(k string) string {
	if v := os.Getenv(k); strings.TrimSpace(v) != "" {
		return v
	}
	return "<none>"
}
func formatOrNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "<none>"
	}
	return s
}
func emoji(b bool) string {
	if b {
		return "✅"
	}
	return "❌"
}
