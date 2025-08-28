package runtime

import (
	"os"
	"strconv"
	"strings"

	"syac/pkg/gitlab"
)

// ShouldUpdateMRDescription gates when we rewrite the MR description with the SYAC block.
// Conditions:
//   - Must be an MR pipeline with a valid MR IID
//   - Target branch must be "dev"
//   - Source branch must match the feature prefix (e.g., "gmarm-")
//   - Allow opt-out via SYAC_UPDATE_MR_DESC=false
func ShouldUpdateMRDescription(c *Context) bool {
	if c == nil || !c.IsMergeRequest || strings.TrimSpace(c.MRID) == "" {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(c.MergeRequestTargetBranch), "dev") {
		return false
	}
	src := strings.ToLower(strings.TrimSpace(firstNonEmpty(c.EffectiveRef, c.RefName)))
	if !strings.HasPrefix(src, strings.ToLower(c.FeatureBranchPrefix)) {
		return false
	}
	// opt-out switch
	return os.Getenv("SYAC_UPDATE_MR_DESC") != "false"
}

// UpsertMRDescriptionIfNeeded is best-effort and never fails the pipeline.
// It inserts the SYAC release-type block into the MR description if missing.
func UpsertMRDescriptionIfNeeded(client *gitlab.Client, c *Context, logger func(string, ...any)) {
	if client == nil || c == nil || !ShouldUpdateMRDescription(c) {
		return
	}
	if c.DryRun {
		logger("[mr] dry-run: would insert SYAC release-type block into MR description !%s", strings.TrimSpace(c.MRID))
		return
	}

	// Prefer known MR IID; otherwise try to resolve by commit SHA.
	mrID := strings.TrimSpace(c.MRID)
	if mrID == "" {
		if mr, err := client.MergeRequests.GetMergeRequestForCommit(c.SHA); err == nil {
			mrID = strconv.Itoa(mr.IID)
		} else {
			logger("[mr] warn: no MRID and lookup by commit failed: %v", err)
			return
		}
	}

	if err := client.MergeRequests.InsertReleaseTypeInDescription(mrID); err != nil {
		logger("[mr] warn: insert description block failed: %v", err) // never fail pipeline
		return
	}
	logger("[mr] inserted SYAC release-type block into MR description on !%s", mrID)
}
