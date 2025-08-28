// internal/docker/build.go
package docker

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"syac/internal/executil"
)

func BuildAndPush(opts *BuildOptions) error {
	if err := BuildImage(opts); err != nil {
		return err
	}
	if opts.Push {
		return PushImage(opts)
	}
	return nil
}

func BuildImage(opts *BuildOptions) error {
	if opts == nil {
		return errors.New("BuildImage: opts is nil")
	}
	if len(opts.FullRefs) == 0 {
		return errors.New("BuildImage: FullRefs must have at least one repo:tag")
	}

	df := strings.TrimSpace(opts.Dockerfile)
	if df == "" {
		df = "Dockerfile"
	}
	ctxPath := strings.TrimSpace(opts.ContextPath)
	if ctxPath == "" {
		ctxPath = "."
	}

	// Only validate filesystem when not in dry-run
	if !opts.DryRun {
		if st, err := os.Stat(df); err != nil || st.IsDir() {
			return fmt.Errorf("BuildImage: Dockerfile %q not found or not a file", df)
		}
		if st, err := os.Stat(ctxPath); err != nil || !st.IsDir() {
			return fmt.Errorf("BuildImage: context %q not found or not a directory", ctxPath)
		}
	}

	dfAbs := absOr(df, df)
	ctxAbs := absOr(ctxPath, ctxPath)

	refs := dedupRefs(opts.FullRefs)
	for _, r := range refs {
		// defensive: Docker tags must be lowercase & no spaces
		if strings.ToLower(r) != r || strings.ContainsAny(r, " \t\n") {
			return fmt.Errorf("BuildImage: invalid ref %q (must be lowercase, no spaces)", r)
		}
	}

	args := []string{"build", "--progress=plain"}
	for _, r := range refs {
		args = append(args, "-t", r)
	}
	args = append(args, "-f", df)
	if opts.Pull {
		args = append(args, "--pull")
	}
	if opts.NoCache {
		args = append(args, "--no-cache")
	}
	if opts.Target != "" {
		args = append(args, "--target", opts.Target)
	}

	// --- sensible default OCI labels (can be overridden by opts.Labels) ---
	// These help provenance in registries and SBOM tools.
	autoLabels := [][2]string{
		{"org.opencontainers.image.revision", getenv("GIT_SHA", "")},
		{"org.opencontainers.image.version", getenv("SYAC_VERSION", getenv("CI_COMMIT_TAG", ""))},
		{"org.opencontainers.image.source", getenv("CI_PROJECT_URL", "")},
		{"org.opencontainers.image.ref.name", getenv("CI_COMMIT_REF_NAME", "")},
	}
	for _, kv := range autoLabels {
		if kv[0] != "" && kv[1] != "" {
			args = append(args, "--label", kv[0]+"="+kv[1])
		}
	}
	for _, kv := range opts.Labels {
		if kv[0] != "" {
			args = append(args, "--label", kv[0]+"="+kv[1])
		}
	}

	for _, kv := range opts.BuildArgs {
		if kv[0] != "" {
			args = append(args, "--build-arg", kv[0]+"="+kv[1])
		}
	}
	args = append(args, ctxPath)

	// Logs
	fmt.Println("— Build Plan —")
	for _, r := range refs {
		fmt.Printf("  tag: %s\n", r)
	}
	fmt.Printf("Dockerfile: %s\n", dfAbs)
	fmt.Printf("Context   : %s\n", ctxAbs)
	fmt.Println("Executing :", "docker", shellQuoteArgs(redactBuildArgs(args)))

	if opts.DryRun {
		return executil.DryRunCMD("docker", args...)
	}
	return executil.RunCMD("docker", args...)
}
