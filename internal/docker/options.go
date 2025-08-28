// internal/docker/options.go
//
// This layer adapts a runtime.Context into concrete BuildOptions
// for the Docker build runner. It pulls in environment overrides,
// resolves the flow, calls the planner for tags/push policy, and
// assembles the minimal set of build args we pass into docker build.
//
// Keep it lean: validation, planner call, build args, return.

package docker

import (
	"fmt"
	"os"
	"strings"

	"syac/internal/runtime"
)

// BuildOptionsFromContext takes the CI runtime context and produces
// a fully-populated BuildOptions struct.
//
// Steps:
//   - validate required context values (registry, app name)
//   - read env overrides (Dockerfile path, context dir)
//   - resolve flow (feature, MR, default, release)
//   - run PlanBuild to decide tags and push policy
//   - prepare standard build args for Dockerfile
func BuildOptionsFromContext(c *runtime.Context) (*BuildOptions, error) {
	if c == nil {
		return nil, fmt.Errorf("nil CI context")
	}
	if strings.TrimSpace(c.RegistryImage) == "" {
		return nil, fmt.Errorf("CI_REGISTRY_IMAGE is empty")
	}
	if strings.TrimSpace(c.ApplicationName) == "" {
		return nil, fmt.Errorf("ApplicationName is empty (set SYAC_APPLICATION_NAME or CI_REGISTRY_IMAGE last segment)")
	}

	// Inputs: Dockerfile + build context (can be overridden via env)
	df := getenv("SYAC_DOCKERFILE", "Dockerfile")
	ctxPath := getenv("SYAC_BUILD_CONTEXT", ".")

	// Resolve flow and generate a build plan (tags + push policy)
	flow := runtime.ResolveFlow(*c, runtime.FlowAuto)
	plan := PlanBuild(*c, flow)
	if len(plan.Refs) == 0 {
		return nil, fmt.Errorf("no image refs produced by planner (flow=%s)", flow)
	}

	// Minimal build args we inject into Dockerfile
	branch := first(c.EffectiveRef, c.RefName)
	args := [][2]string{
		{"GIT_SHA", c.SHA},
		{"GIT_SHORT_SHA", c.ShortSHA},
		{"CI_PROJECT_PATH", c.ProjectPath},
		{"CI_REF_NAME", branch},
		{"APP_NAME", c.ApplicationName},
	}

	// Return BuildOptions which downstream build.go consumes
	return &BuildOptions{
		Dockerfile:  df,
		ContextPath: ctxPath,
		BuildArgs:   args,
		FullRefs:    plan.Refs, // tags from planner
		Pull:        os.Getenv("SYAC_PULL") == "true",
		NoCache:     os.Getenv("SYAC_NOCACHE") == "true",
		Push:        plan.Push, // push flag from planner
		DryRun:      os.Getenv("SYAC_DRY_RUN") == "true",
	}, nil
}
