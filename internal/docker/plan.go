// internal/docker/plan.go
//
// The planner converts a runtime.Context + resolved Flow into a Plan
// (image tags + push policy). This is the "brains" of release behavior.
//
// Rules (semver-first RC):
//   - feature  → :<shortsha> [ + :latest if SYAC_LATEST_ON_FEATURE=true ]
//                 push only if PUSH_FEATURE=true
//   - mr       → :<shortsha>, :<next-rc-with-shortsha> (always push)
//   - default  → :<shortsha>, :<next-rc-with-shortsha>, :<branch>
//                 [ + :latest if SYAC_LATEST_ON_DEFAULT=true ]
//   - release  → :<tag> [ + :latest if SYAC_TAG_LATEST=true ]
//
// This keeps policy isolated and testable; BuildOptionsFromContext
// just calls into here.

package docker

import (
	"fmt"
	"os"
	"strings"

	"syac/internal/runtime"
)

// Plan is the output of the planner: tags + push flag.
type Plan struct {
	Refs []string // fully-qualified repo:tag
	Push bool     // whether we should push after build
}

// PlanBuild turns Context + Flow into a Plan (tags + push policy).
func PlanBuild(ctx runtime.Context, flow runtime.Flow) Plan {
	baseImg := strings.TrimSpace(ctx.RegistryImage)
	app := strings.TrimSpace(ctx.ApplicationName)
	if baseImg == "" || app == "" {
		// Fail-safe: no base image/app to tag. Caller should treat as error.
		return Plan{Refs: nil, Push: false}
	}

	base := strings.TrimRight(baseImg, "/") + "/" + app
	add := func(out *[]string, tag string) {
		tag = cleanTag(tag)
		if tag == "" || !validateTag(tag) {
			return
		}
		*out = append(*out, fmt.Sprintf("%s:%s", base, tag))
	}

	var refs []string

	switch flow {
	case runtime.FlowFeature:
		// Always tag with short SHA
		add(&refs, ctx.ShortSHA)
		// Optional "latest" on feature branch (explicit opt-in)
		if os.Getenv("SYAC_LATEST_ON_FEATURE") == "true" {
			add(&refs, "latest")
		}

	case runtime.FlowMR:
		// Short SHA + semver-based RC
		add(&refs, ctx.ShortSHA)
		add(&refs, ctx.NextRCVersion)

	case runtime.FlowDefault:
		// Default branch: short SHA, RC, and channel tag
		add(&refs, ctx.ShortSHA)
		add(&refs, ctx.NextRCVersion)
		add(&refs, ctx.DefaultBranch)
		// Optional "latest" on default branch
		if os.Getenv("SYAC_LATEST_ON_DEFAULT") == "true" {
			add(&refs, "latest")
		}

	case runtime.FlowRelease:
		// Final release tag
		add(&refs, ctx.Tag)
		// Optional "latest" on release (common practice)
		if os.Getenv("SYAC_TAG_LATEST") == "true" {
			add(&refs, "latest")
		}

	default:
		// Fallback: behave like default branch
		add(&refs, ctx.ShortSHA)
		add(&refs, ctx.NextRCVersion)
		if ctx.IsDefaultBranch && ctx.DefaultBranch != "" {
			add(&refs, ctx.DefaultBranch)
		}
	}

	// Deduplicate to keep tags clean and deterministic
	refs = dedupRefs(refs)

	// Push policy: features are gated, everything else always pushes
	push := true
	if flow == runtime.FlowFeature && strings.EqualFold(strings.TrimSpace(ctx.Source), "push") {
		push = getenv("PUSH_FEATURE", "") == "true"
	}

	return Plan{Refs: refs, Push: push}
}
