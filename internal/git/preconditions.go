package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnsureGitReady checks git availability, finds or inits a repo,
// and validates that all target dirs are within the repo root.
// Returns the absolute repo root path.
func EnsureGitReady(dirs []string) (string, error) {
	// Step 1: Check git is available
	if _, err := exec.LookPath("git"); err != nil {
		return "", fmt.Errorf("muti requires git. Install git and try again")
	}

	// Configure git for automated environments (like Docker)
	_ = exec.Command("git", "config", "--global", "--add", "safe.directory", "*").Run()
	_ = exec.Command("git", "config", "--global", "user.name", "muti").Run()
	_ = exec.Command("git", "config", "--global", "user.email", "muti@localhost").Run()

	// Step 2: Find or init repo
	repoRoot, err := findRepoRoot()
	if err != nil {
		// Not a git repo — initialize one
		repoRoot, err = initRepo()
		if err != nil {
			return "", fmt.Errorf("failed to initialize git repo: %w", err)
		}
		fmt.Fprintln(os.Stderr, "⚠ Initialized temporary git repo")
	}

	// Step 3: Validate all dirs are within repo root
	for _, d := range dirs {
		absDir, err := filepath.Abs(d)
		if err != nil {
			return "", fmt.Errorf("cannot resolve directory %q: %w", d, err)
		}
		rel, err := filepath.Rel(repoRoot, absDir)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("directory %q is outside git repo root %q. All target directories must be within the same git repository", d, repoRoot)
		}
	}

	// Step 4: Ensure at least one commit exists
	if err := ensureCommit(repoRoot); err != nil {
		return "", fmt.Errorf("failed to create initial commit: %w", err)
	}

	return repoRoot, nil
}

func findRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func initRepo() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	cmds := [][]string{
		{"git", "init"},
		{"git", "add", "-A"},
		{"git", "commit", "-m", "muti: initial snapshot", "--allow-empty"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = cwd
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("command %v failed: %w", args, err)
		}
	}
	return cwd, nil
}

func ensureCommit(repoRoot string) error {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoRoot
	if err := cmd.Run(); err != nil {
		// No commits yet
		cmds := [][]string{
			{"git", "add", "-A"},
			{"git", "commit", "-m", "muti: initial snapshot", "--allow-empty"},
		}
		for _, args := range cmds {
			c := exec.Command(args[0], args[1:]...)
			c.Dir = repoRoot
			if e := c.Run(); e != nil {
				return e
			}
		}
	}
	return nil
}

// ResetDirs resets the given directories to their original state via git checkout.
func ResetDirs(dir string, dirs []string) {
	args := append([]string{"checkout", "--"}, dirs...)
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	_ = cmd.Run()
}
