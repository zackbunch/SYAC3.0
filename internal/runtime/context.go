package runtime

import (
	"fmt"
	"os"
	"strings"

	"syac/internal/version"
	"syac/pkg/gitlab"
)

// Context captures the relevant CI/CD environment state for SYAC.
// This assumes execution inside GitLab CI.
type Context struct {
	Source                   string
	RefName                  string
	EffectiveRef             string
	SHA                      string
	ShortSHA                 string
	MRID                     string
	Tag                      string
	ProjectPath              string
	RegistryImage            string
	DefaultBranch            string
	Sprint                   string
	ApplicationName          string
	MergeRequestTargetBranch string
	ProjectID                string

	// Derived booleans
	IsMergeRequest      bool
	IsTag               bool
	IsFeatureBranch     bool
	FeatureBranchPrefix string
	IsDefaultBranch     bool
	DryRun              bool

	// Proposed release metadata
	// For feature branches, this is ALWAYS the short SHA.
	FeatureTag string
	ImageRef   string

	// Version forecast metadata
	BumpType      version.VersionType // Major | Minor | Patch
	NextVersion   string              // e.g., "1.4.3" or "v1.4.3"
	NextRCVersion string              // e.g., "1.4.3-rc.1" or "v1.4.3-<shortsha>"
}

// LoadContext constructs a CI Context by reading GitLab CI/CD environment variables.
func LoadContext() (Context, error) {
	tag := os.Getenv("CI_COMMIT_TAG")
	def := os.Getenv("CI_DEFAULT_BRANCH")

	effectiveRef := firstNonEmpty(
		os.Getenv("CI_MERGE_REQUEST_SOURCE_BRANCH_NAME"),
		os.Getenv("CI_COMMIT_BRANCH"),
		os.Getenv("CI_COMMIT_REF_NAME"),
	)
	rawRef := strings.TrimSpace(os.Getenv("CI_COMMIT_REF_NAME"))

	const featurePrefix = "gmarm-"

	isMR := os.Getenv("CI_MERGE_REQUEST_IID") != "" ||
		os.Getenv("CI_PIPELINE_SOURCE") == "merge_request_event"
	isTag := tag != ""

	isFeature := !isTag &&
		effectiveRef != "" &&
		effectiveRef != def &&
		strings.HasPrefix(effectiveRef, featurePrefix)

	// Ensure ShortSHA is populated (fallback if CI_COMMIT_SHORT_SHA is missing)
	short := os.Getenv("CI_COMMIT_SHORT_SHA")
	if short == "" {
		sha := os.Getenv("CI_COMMIT_SHA")
		if len(sha) >= 8 {
			short = sha[:8]
		} else {
			short = sha
		}
	}

	// Determine bump type: env override or default Patch
	bump := version.Patch
	if v := strings.TrimSpace(os.Getenv("SYAC_BUMP")); v != "" {
		if vt, err := version.ParseVersionType(v); err == nil {
			bump = vt
		}
	}

	ctx := Context{
		Source:                   os.Getenv("CI_PIPELINE_SOURCE"),
		RefName:                  rawRef,
		EffectiveRef:             effectiveRef,
		SHA:                      os.Getenv("CI_COMMIT_SHA"),
		ShortSHA:                 short,
		MRID:                     os.Getenv("CI_MERGE_REQUEST_IID"),
		MergeRequestTargetBranch: os.Getenv("CI_MERGE_REQUEST_TARGET_BRANCH_NAME"),
		Tag:                      tag,
		ProjectPath:              os.Getenv("CI_PROJECT_PATH"),
		RegistryImage:            os.Getenv("CI_REGISTRY_IMAGE"),
		DefaultBranch:            def,
		Sprint:                   os.Getenv("SYAC_SPRINT"),
		IsMergeRequest:           isMR,
		IsTag:                    isTag,
		IsFeatureBranch:          isFeature,
		FeatureBranchPrefix:      featurePrefix,
		IsDefaultBranch:          rawRef != "" && rawRef == def,
		ProjectID:                os.Getenv("CI_PROJECT_ID"),
		ApplicationName:          resolveApplicationName(),
		DryRun:                   os.Getenv("SYAC_DRY_RUN") == "true",
		BumpType:                 bump,
	}

	// Feature branches: only short SHA as tag.
	if ctx.IsFeatureBranch {
		ctx.FeatureTag = ctx.ShortSHA
	}

	// Build ImageRef:
	//   - feature branch -> <image>/<app>:<shortsha>
	//   - tag pipeline   -> <image>/<app>:<tag>
	//   - default branch -> <image>/<app>:<default-branch>
	if ctx.RegistryImage != "" && ctx.ApplicationName != "" {
		base := fmt.Sprintf("%s/%s", ctx.RegistryImage, ctx.ApplicationName)
		switch {
		case ctx.IsFeatureBranch && ctx.FeatureTag != "":
			ctx.ImageRef = fmt.Sprintf("%s:%s", base, ctx.FeatureTag)
		case ctx.IsTag && ctx.Tag != "":
			ctx.ImageRef = fmt.Sprintf("%s:%s", base, ctx.Tag)
		case ctx.IsDefaultBranch && ctx.DefaultBranch != "":
			ctx.ImageRef = fmt.Sprintf("%s:%s", base, ctx.DefaultBranch)
		}
	}

	return ctx, nil
}

// resolveApplicationName picks the application name:
// 1. If SYAC_APPLICATION_NAME is set, use that.
// 2. Otherwise, fall back to the last segment of CI_REGISTRY_IMAGE.
func resolveApplicationName() string {
	if v := os.Getenv("SYAC_APPLICATION_NAME"); v != "" {
		return v
	}
	registryImage := os.Getenv("CI_REGISTRY_IMAGE")
	parts := strings.Split(strings.TrimSpace(registryImage), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// PrintSummary emits a scannable CI/CD context report with logical sections.
// NOTE: pointer receiver so computed fields (e.g., NextRCVersion) persist.
func (c *Context) PrintSummary(client *gitlab.Client) {
	fmt.Println("CI/CD Environment Summary")
	fmt.Println("--------------------------")

	// ── Pipeline / Job ───────────────────────────────────────────────────────────
	fmt.Println("Pipeline")
	fmt.Printf("  Context               : %s\n", c.describeContext())
	fmt.Printf("  Pipeline Source       : %s\n", c.Source)
	fmt.Printf("  Pipeline ID           : %s\n", getEnvOrNone("CI_PIPELINE_ID"))
	fmt.Printf("  Pipeline URL          : %s\n", getEnvOrNone("CI_PIPELINE_URL"))
	fmt.Printf("  Job ID                : %s\n", getEnvOrNone("CI_JOB_ID"))
	fmt.Println()

	// ── Ref / Commit ────────────────────────────────────────────────────────────
	fmt.Println("Ref / Commit")
	fmt.Printf("  Branch or Tag (raw)   : %s\n", formatOrNone(c.RefName))
	fmt.Printf("  Effective Ref         : %s\n", formatOrNone(c.EffectiveRef))
	if c.IsTag {
		fmt.Printf("  Tag                   : %s\n", c.Tag)
	}
	fmt.Printf("  Default Branch        : %s\n", formatOrNone(c.DefaultBranch))
	fmt.Printf("  Feature Prefix        : %s\n", c.FeatureBranchPrefix)
	fmt.Printf("  Commit SHA            : %s\n", c.SHA)
	fmt.Printf("  Commit Short SHA      : %s\n", c.ShortSHA)
	if c.FeatureTag != "" {
		fmt.Printf("  Proposed Feature Tag  : %s\n", c.FeatureTag)
	}
	fmt.Println()

	// ── Merge Request (conditional) ─────────────────────────────────────────────
	if c.IsMergeRequest {
		fmt.Println("Merge Request")
		fmt.Printf("  Merge Request IID     : %s\n", formatOrNone(c.MRID))
		fmt.Printf("  Target Branch         : %s\n", formatOrNone(c.MergeRequestTargetBranch))
		fmt.Println()
	}

	// ── Project / Image ─────────────────────────────────────────────────────────
	fmt.Println("Project")
	fmt.Printf("  Project Path          : %s\n", c.ProjectPath)
	fmt.Printf("  Project ID            : %s\n", c.ProjectID)
	fmt.Printf("  Registry Image        : %s\n", c.RegistryImage)
	if c.ImageRef != "" {
		fmt.Printf("  Image Ref             : %s\n", c.ImageRef)
	}
	fmt.Printf("  Application Name      : %s\n", formatOrNone(c.ApplicationName))
	fmt.Println()

	// ── Derived Flags ───────────────────────────────────────────────────────────
	fmt.Println("Derived")
	fmt.Printf("  Is Merge Request      : %s\n", emoji(c.IsMergeRequest))
	fmt.Printf("  Is Default Branch     : %s\n", emoji(c.IsDefaultBranch))
	fmt.Printf("  Is Feature Branch     : %s\n", emoji(c.IsFeatureBranch))
	fmt.Printf("  Is Tag Build          : %s\n", emoji(c.IsTag))
	fmt.Printf("  Dry Run Mode          : %s\n", emoji(c.DryRun))
	// If this is an MR and no explicit SYAC_BUMP was provided, try to pull from the MR.
	// Do this BEFORE printing the bump type so we only print once.
	if src := c.ResolveBumpFromMR(client); src != "" {
		fmt.Printf("  Bump Type             : %s (from %s)\n", c.BumpType.String(), src)
	} else {
		fmt.Printf("  Bump Type             : %s\n", c.BumpType.String())
	}
	fmt.Println()

	// ── Tags + Forecast ─────────────────────────────────────────────────────────
	fmt.Println("Tags")

	var latestTagStr string

	if client == nil {
		fmt.Println("  Status                : Skipped (no GitLab client)")
	} else {
		// Use Tags service as the single source of truth.
		// This already defaults to 0.0.0 when no valid semver tags exist.
		current, next, err := client.Tags.GetNextVersion(c.BumpType)
		if err != nil {
			fmt.Printf("  Status                : Error (%v)\n", err)
		} else {
			latestTagStr = current.String()
			c.NextVersion = next.String()

			fmt.Printf("  Latest Tag            : %s\n", latestTagStr)
			fmt.Printf("  Next Version          : %s\n", c.NextVersion)

			// Compute RC tag (semver-next + short SHA) for MR/dev flows.
			if (c.IsMergeRequest || c.IsDefaultBranch) && c.ShortSHA != "" {
				if rc, rerr := version.ForecastNextRC(latestTagStr, c.BumpType, c.ShortSHA); rerr == nil {
					c.NextRCVersion = rc
					fmt.Printf("  Next RC Version       : %s\n", c.NextRCVersion)
				} else {
					fmt.Printf("  Next RC Version       : (error) %v\n", rerr)
				}
			}
		}
	}
	fmt.Println()
}

func (c Context) describeContext() string {
	switch {
	case c.IsMergeRequest:
		return "Merge Request"
	case c.IsTag:
		return fmt.Sprintf("Tag push (%s)", c.Tag)
	case c.IsDefaultBranch:
		return fmt.Sprintf("Push to default branch (%s)", c.RefName)
	case c.IsFeatureBranch:
		return fmt.Sprintf("Development Branch (%s)", c.EffectiveRef)
	}
	if c.EffectiveRef != "" {
		return fmt.Sprintf("Branch push (%s)", c.EffectiveRef)
	}
	return fmt.Sprintf("Pipeline source: %s", c.Source)
}

// ResolveBumpFromMR overrides BumpType from the MR checkbox selection when possible.
// - Only applies on merge request pipelines with a valid MRID
// - Respects env override: if SYAC_BUMP is set, we do nothing.
// Returns a short string describing where the bump came from, or "" if unchanged.
func (c *Context) ResolveBumpFromMR(client *gitlab.Client) string {
	// If user explicitly set SYAC_BUMP, don't second-guess them.
	if strings.TrimSpace(os.Getenv("SYAC_BUMP")) != "" {
		return ""
	}
	// Need an MR and a client to query.
	if !c.IsMergeRequest || strings.TrimSpace(c.MRID) == "" || client == nil {
		return ""
	}

	bump, err := client.MergeRequests.GetVersionBump(c.MRID)
	if err != nil {
		// Keep existing BumpType on error.
		return ""
	}

	// Only log a change if it actually changes the value.
	if bump != c.BumpType {
		c.BumpType = bump
		return "MR selection"
	}
	return "MR selection"
}
