package app

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

const defaultGitRemote = "origin"
const changeBranchPrefix = "change/"

var branchZeroPaddingPattern = regexp.MustCompile(`/0+`)

type gitCommandRunner func(ctx context.Context, dir string, args ...string) (string, error)

var referenceGitRunner gitCommandRunner = runGitReferenceCommand

func runGitReferenceCommand(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(output))
	if err != nil {
		if text != "" {
			return text, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, text)
		}
		return text, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return text, nil
}

func reconcileChangeBranch(ref string, slug string) error {
	return reconcileChangeBranchWithRunner(context.Background(), referenceGitRunner, strings.TrimSpace(ref), strings.TrimSpace(slug))
}

func reconcileChangeBranchWithRunner(ctx context.Context, run gitCommandRunner, ref string, slug string) error {
	if ref == "" {
		return fmt.Errorf("change ref is required")
	}
	if slug == "" {
		return fmt.Errorf("change slug is required")
	}
	repoRoot, err := gitOutput(ctx, run, "", "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("git repository is required: %w", err)
	}
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return fmt.Errorf("git repository root is required")
	}

	branch := changeBranchPrefix + slug
	if localBranch(ctx, run, repoRoot, branch) != "" {
		_, err := gitOutput(ctx, run, repoRoot, "checkout", branch)
		return err
	}
	if oldLocal := firstLocalChangeBranchForRef(ctx, run, repoRoot, ref); oldLocal != "" {
		if _, err := gitOutput(ctx, run, repoRoot, "checkout", oldLocal); err != nil {
			return err
		}
		_, err := gitOutput(ctx, run, repoRoot, "branch", "-m", branch)
		return err
	}
	if remoteBranch(ctx, run, repoRoot, branch) != "" {
		if err := fetchRemoteBranch(ctx, run, repoRoot, branch); err != nil {
			return err
		}
		_, err := gitOutput(ctx, run, repoRoot, "checkout", "-b", branch, "--track", defaultGitRemote+"/"+branch)
		return err
	}
	if oldRemote := firstRemoteChangeBranchForRef(ctx, run, repoRoot, ref); oldRemote != "" {
		if err := fetchRemoteBranch(ctx, run, repoRoot, oldRemote); err != nil {
			return err
		}
		if _, err := gitOutput(ctx, run, repoRoot, "checkout", "-b", oldRemote, "--track", defaultGitRemote+"/"+oldRemote); err != nil {
			return err
		}
		if _, err := gitOutput(ctx, run, repoRoot, "branch", "-m", branch); err != nil {
			return err
		}
		if _, err := gitOutput(ctx, run, repoRoot, "push", defaultGitRemote, branch); err != nil {
			return err
		}
		_, err := gitOutput(ctx, run, repoRoot, "push", defaultGitRemote, "--delete", oldRemote)
		return err
	}
	_, err = gitOutput(ctx, run, repoRoot, "checkout", "-b", branch)
	return err
}

func gitOutput(ctx context.Context, run gitCommandRunner, dir string, args ...string) (string, error) {
	output, err := run(ctx, dir, args...)
	return strings.TrimSpace(output), err
}

func localBranch(ctx context.Context, run gitCommandRunner, repoRoot string, branch string) string {
	return firstLocalBranch(ctx, run, repoRoot, branch)
}

func firstLocalBranch(ctx context.Context, run gitCommandRunner, repoRoot string, pattern string) string {
	output, err := gitOutput(ctx, run, repoRoot, "branch", "--list", "--format=%(refname:short)", pattern)
	if err != nil {
		return ""
	}
	return firstNonEmptyLine(output)
}

func firstLocalChangeBranchForRef(ctx context.Context, run gitCommandRunner, repoRoot string, ref string) string {
	output, err := gitOutput(ctx, run, repoRoot, "branch", "--list", "--format=%(refname:short)", changeBranchPrefix+"*")
	if err != nil {
		return ""
	}
	return firstBranchMatchingRef(output, ref)
}

func remoteBranch(ctx context.Context, run gitCommandRunner, repoRoot string, branch string) string {
	return firstRemoteBranch(ctx, run, repoRoot, branch)
}

func fetchRemoteBranch(ctx context.Context, run gitCommandRunner, repoRoot string, branch string) error {
	_, err := gitOutput(ctx, run, repoRoot, "fetch", defaultGitRemote, "refs/heads/"+branch+":refs/remotes/"+defaultGitRemote+"/"+branch)
	return err
}

func firstRemoteChangeBranchForRef(ctx context.Context, run gitCommandRunner, repoRoot string, ref string) string {
	output, err := gitOutput(ctx, run, repoRoot, "ls-remote", "--heads", defaultGitRemote, "refs/heads/"+changeBranchPrefix+"*")
	if err != nil {
		return ""
	}
	var branches []string
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		branch, ok := strings.CutPrefix(fields[1], "refs/heads/")
		if ok {
			branches = append(branches, branch)
		}
	}
	return firstBranchMatchingRef(strings.Join(branches, "\n"), ref)
}

func firstRemoteBranch(ctx context.Context, run gitCommandRunner, repoRoot string, pattern string) string {
	output, err := gitOutput(ctx, run, repoRoot, "ls-remote", "--heads", defaultGitRemote, "refs/heads/"+pattern)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		branch, ok := strings.CutPrefix(fields[1], "refs/heads/")
		if ok && strings.TrimSpace(branch) != "" {
			return strings.TrimSpace(branch)
		}
	}
	return ""
}

func firstBranchMatchingRef(output string, ref string) string {
	matchPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(changeBranchPrefix) + regexp.QuoteMeta(ref) + `-`)
	for _, line := range strings.Split(output, "\n") {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branch == "" {
			continue
		}
		normalized := branchZeroPaddingPattern.ReplaceAllString(branch, "/")
		if matchPattern.MatchString(normalized) {
			return branch
		}
	}
	return ""
}

func firstNonEmptyLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if line != "" {
			return line
		}
	}
	return ""
}
