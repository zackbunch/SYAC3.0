// internal/executil/executil.go
package executil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// RunCMD executes the given command with inherited stdout/stderr.
func RunCMD(name string, args ...string) error {
	return runCore(context.Background(), "", nil, false, name, args...)
}

// RunCMDWithDir executes the command in a specific directory.
func RunCMDWithDir(dir, name string, args ...string) error {
	return runCore(context.Background(), dir, nil, false, name, args...)
}

// DryRunCMD logs the command that would be run without executing.
func DryRunCMD(name string, args ...string) error {
	return runCore(context.Background(), "", nil, true, name, args...)
}

// DryRunCMDWithDir logs the command that would be run in a specific directory.
func DryRunCMDWithDir(dir, name string, args ...string) error {
	return runCore(context.Background(), dir, nil, true, name, args...)
}

// ---- Power-user variants (optional but handy) ----

// RunCtx executes with a context (for timeouts/cancellation).
func RunCtx(ctx context.Context, name string, args ...string) error {
	return runCore(ctx, "", nil, false, name, args...)
}

// RunWithEnv executes with additional environment variables.
func RunWithEnv(dir string, extraEnv map[string]string, name string, args ...string) error {
	return runCore(context.Background(), dir, extraEnv, false, name, args...)
}

// DryRunWithEnv logs a dry-run with extra env and dir.
func DryRunWithEnv(dir string, extraEnv map[string]string, name string, args ...string) error {
	return runCore(context.Background(), dir, extraEnv, true, name, args...)
}

// ----------------------------------------------------------------

func runCore(ctx context.Context, dir string, extraEnv map[string]string, dry bool, name string, args ...string) error {
	fullCmd := name + " " + shellQuoteArgs(args)
	prefix := ""
	if dir != "" {
		prefix = " in " + dir
	}

	if dry {
		if dir != "" {
			fmt.Printf("[DRY RUN%s] %s\n", prefix, fullCmd)
		} else {
			fmt.Printf("[DRY RUN] %s\n", fullCmd)
		}
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	// apply extra env if provided
	if len(extraEnv) > 0 {
		for k, v := range extraEnv {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	fmt.Printf("Running%s: %s\n", prefix, fullCmd)
	if err := cmd.Run(); err != nil {
		// include exit status if available
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return fmt.Errorf("command failed (exit=%d): %s: %w", status.ExitStatus(), fullCmd, err)
			}
		}
		// context cancellations/timeouts show clearly
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("command canceled: %s", fullCmd)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("command timed out: %s", fullCmd)
		}
		return fmt.Errorf("failed to run command: %s: %w", fullCmd, err)
	}
	return nil
}

// shellQuoteArgs returns a printable, shell-safe representation of args.
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

// Timeout helper if you want one-liners like executil.RunWithTimeout(...)
func RunWithTimeout(timeout time.Duration, dir, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return runCore(ctx, dir, nil, false, name, args...)
}
