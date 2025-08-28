// internal/docker/push.go
//
// Handles pushing built Docker images to the GitLab registry.
// - Reads CI_REGISTRY / CI_REGISTRY_USER / CI_REGISTRY_PASSWORD (or CI_JOB_TOKEN).
// - Logs in, pushes each tag, logs out.
// - Respects DryRun mode: prints commands instead of executing.
//
// Keep this file focused only on the registry side of the flow.
// Building/tagging is handled elsewhere.

package docker

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"syac/internal/executil"
)

// PushImage logs into the GitLab registry and pushes every ref in opts.FullRefs.
// It respects opts.DryRun (commands are printed, not executed).
func PushImage(opts *BuildOptions) error {
	if opts == nil {
		return errors.New("PushImage: opts is nil")
	}
	refs := dedupRefs(opts.FullRefs)
	if len(refs) == 0 {
		return errors.New("PushImage: no refs to push (FullRefs empty)")
	}

	// Pull creds from environment (CI-provided)
	registry, user, password := credsFromEnv()
	if registry == "" || user == "" {
		return fmt.Errorf("missing CI_REGISTRY or CI_REGISTRY_USER")
	}
	if password == "" {
		return fmt.Errorf("missing CI_REGISTRY_PASSWORD or CI_JOB_TOKEN")
	}

	// Docker login
	if err := login(registry, user, password, opts.DryRun); err != nil {
		return fmt.Errorf("docker login failed: %w", err)
	}
	if !opts.DryRun {
		// Only log out if we actually logged in
		defer logout(registry)
	}

	// Push each tag
	for _, r := range refs {
		if err := pushRef(r, opts.DryRun); err != nil {
			return err
		}
	}
	return nil
}

// credsFromEnv pulls registry/user/password from GitLab CI variables.
func credsFromEnv() (registry, user, password string) {
	registry = os.Getenv("CI_REGISTRY")
	user = os.Getenv("CI_REGISTRY_USER")
	password = os.Getenv("CI_REGISTRY_PASSWORD")
	if password == "" {
		password = os.Getenv("CI_JOB_TOKEN")
	}
	return
}

// login runs a docker login (masked if dry-run).
func login(registry, user, password string, dry bool) error {
	if dry {
		return executil.DryRunCMD("docker", "login", "-u", user, "-p", "[REDACTED]", registry)
	}
	return executil.RunCMD("docker", "login", "-u", user, "-p", password, registry)
}

// logout runs docker logout, but doesn’t fail the pipeline if it errors.
func logout(registry string) {
	if err := executil.RunCMD("docker", "logout", registry); err != nil {
		fmt.Fprintf(os.Stderr, "warning: docker logout failed: %v\n", err)
	}
}

// pushRef pushes a single tag (respects dry-run).
func pushRef(ref string, dry bool) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil
	}
	if dry {
		return executil.DryRunCMD("docker", "push", ref)
	}
	fmt.Printf("Pushing image: %s\n", ref)
	return executil.RunCMD("docker", "push", ref)
}
