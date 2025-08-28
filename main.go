// syac main entrypoint
//
// This binary is meant to run inside GitLab CI as a single build stage.
// It detects the pipeline context (feature, MR, default branch, release),
// figures out the right image tags, posts MR comments when needed, and
// builds/pushes Docker images accordingly.
//
// Keep this file simple: load context, annotate (best-effort), print summary,
// resolve flow, build options, build/push. All the heavy lifting stays internal.

package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"syac/internal/docker"
	"syac/internal/runtime"
	"syac/pkg/gitlab"
)

func main() {
	// Local overrides for dev runs; harmless in CI.
	_ = godotenv.Load("environments/mr.env")

	// 1) CI/CD runtime context
	ctx, err := runtime.LoadContext()
	if err != nil {
		log.Fatalf("failed to load context: %v", err)
	}

	// 2) GitLab client
	client, err := gitlab.NewClient()
	if err != nil {
		log.Printf("[gitlab] init failed; skipping MR annotate + release lookup: %v", err)
	} else {
		// 2a) Early, best-effort MR annotate (idempotent). Non-blocking by design.
		runtime.UpsertMRDescriptionIfNeeded(client, &ctx, log.Printf)
	}

	// 3) Print summary (does MR bump resolution + version forecast)
	(&ctx).PrintSummary(client) // safe with nil client; guards inside

	// 4) Resolve flow → tags/push policy are derived from it
	flow := runtime.ResolveFlow(ctx, runtime.FlowAuto)
	log.Printf("[syac] resolved flow: %s", flow)

	// 5) Build options (refs, build args, push flag)
	opts, err := docker.BuildOptionsFromContext(&ctx)
	if err != nil {
		log.Fatalf("failed to create build options: %v", err)
	}

	// 6) Debug what we'll actually do
	log.Printf("[docker] refs: %v", opts.FullRefs)
	log.Printf("[docker] push=%v (SYAC_PUSH=%q, PUSH_FEATURE=%q, source=%q, branch=%q)",
		opts.Push,
		os.Getenv("SYAC_PUSH"),
		os.Getenv("PUSH_FEATURE"),
		ctx.Source,
		func() string {
			if ctx.EffectiveRef != "" {
				return ctx.EffectiveRef
			}
			return ctx.RefName
		}(),
	)

	// 7) Build (and push if enabled). Honors dry-run.
	if err := docker.BuildAndPush(opts); err != nil {
		log.Fatalf("build/push failed: %v", err)
	}
}
