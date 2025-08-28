package docker

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ---- FS / shell helpers ----

func absOr(p, fallback string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return fallback
}

func shellQuoteArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\n\"'`$\\*?[]{}()<>|&;") {
			a = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
		quoted[i] = a
	}
	return strings.Join(quoted, " ")
}

// ---- Env / redaction ----

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func redactBuildArgs(args []string) []string {
	// broaden secret heuristics
	sus := func(k string) bool {
		k = strings.ToUpper(k)
		return strings.Contains(k, "PASSWORD") ||
			strings.Contains(k, "TOKEN") ||
			strings.Contains(k, "SECRET") ||
			k == "CI_JOB_TOKEN" ||
			k == "DOCKER_AUTH_CONFIG" ||
			k == "AWS_SECRET_ACCESS_KEY" ||
			k == "AWS_SESSION_TOKEN" ||
			k == "GITHUB_TOKEN" || k == "GH_TOKEN" ||
			k == "GOOGLE_APPLICATION_CREDENTIALS" ||
			k == "KUBECONFIG"
	}
	out := make([]string, len(args))
	copy(out, args)
	for i := 0; i < len(out)-1; i++ {
		if out[i] == "--build-arg" {
			kv := out[i+1]
			if eq := strings.IndexByte(kv, '='); eq > 0 {
				key := kv[:eq]
				val := kv[eq+1:]
				if sus(key) && val != "" {
					out[i+1] = key + "=REDACTED"
				}
			}
		}
	}
	return out
}

// ---- Tag normalization / validation ----

var tagAllowed = regexp.MustCompile(`^[a-z0-9_.-]{1,128}$`)

func cleanTag(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	// normalize common offenders early
	repl := []struct{ from, to string }{
		{"/", "-"},
		{" ", "-"},
	}
	for _, r := range repl {
		s = strings.ReplaceAll(s, r.from, r.to)
	}
	// collapse multiple hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	// trim to Docker's max tag length
	if len(s) > 128 {
		s = s[:128]
	}
	return s
}

func validateTag(tag string) bool {
	return tagAllowed.MatchString(tag)
}

// dedupRefs preserves insertion order.
func dedupRefs(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	// deterministic if you ever append sets (not needed now, keep import quiet)
	_ = sort.Strings
	return out
}

// first non-empty
func first(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
