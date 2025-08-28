package version

import (
	"fmt"
	"strconv"
	"strings"
)

func (vt VersionType) String() string {
	return string(vt)
}

// VersionType represents a semantic version bump level
type VersionType string

const (
	Patch VersionType = "Patch"
	Minor VersionType = "Minor"
	Major VersionType = "Major"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Parse parses a version string in the format "X.Y.Z"
func Parse(versionStr string) (Version, error) {
	parts := strings.Split(versionStr, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version format: expected X.Y.Z, got %s", versionStr)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %w", err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version: %w", err)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version: %w", err)
	}

	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

// Inc returns a new Version incremented based on the given semantic version type
func (v Version) Increment(bump VersionType) Version {
	switch bump {
	case Major:
		return Version{Major: v.Major + 1, Minor: 0, Patch: 0}
	case Minor:
		return Version{Major: v.Major, Minor: v.Minor + 1, Patch: 0}
	case Patch:
		return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	default:
		return v // no change if invalid bump type
	}
}

func (v Version) LessThan(other Version) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	return v.Patch < other.Patch
}

// ParseVersionType converts a string like "major" into a VersionType enum
func ParseVersionType(s string) (VersionType, error) {
	switch strings.ToLower(s) {
	case "major":
		return Major, nil
	case "minor":
		return Minor, nil
	case "patch":
		return Patch, nil
	default:
		return "", fmt.Errorf("invalid bump type: %q. Must be one of: major, minor, patch", s)
	}
}

// ForecastNext takes the latest tag (e.g., "v1.2.3" or "1.2.3")
// and the desired bump, and returns the next version preserving any "v" prefix.
func ForecastNext(latestTag string, bump VersionType) (string, error) {
	latestTag = strings.TrimSpace(latestTag)

	// Preserve prefix style
	hasV := strings.HasPrefix(latestTag, "v")
	core := latestTag
	if hasV {
		core = strings.TrimPrefix(latestTag, "v")
	}

	// No latest -> treat as 0.0.0 and bump
	if core == "" {
		base := (Version{Major: 0, Minor: 0, Patch: 0}).Increment(bump)
		if hasV {
			return "v" + base.String(), nil
		}
		return base.String(), nil
	}

	// Parse X.Y.Z and bump
	v, err := Parse(core)
	if err != nil {
		return "", fmt.Errorf("unable to parse latest tag %q: %w", latestTag, err)
	}
	next := v.Increment(bump).String()
	if hasV {
		return "v" + next, nil
	}
	return next, nil
}

// ForecastNextRC is like ForecastNext but appends a pre-release identifier.
// Example: v1.2.4 -> v1.2.4-rc.1
// If suffix == "", defaults to "rc.1".
func ForecastNextRC(latestTag string, bump VersionType, suffix string) (string, error) {
	next, err := ForecastNext(latestTag, bump)
	if err != nil {
		return "", err
	}
	if suffix == "" {
		suffix = "rc.1"
	}
	return fmt.Sprintf("%s-%s", next, suffix), nil
}
