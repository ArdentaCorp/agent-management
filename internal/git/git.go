package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// URLInfo holds the parsed components of a GitHub URL.
type URLInfo struct {
	URL    string // normalized .git URL
	Branch string // branch name (if specified in URL)
	Path   string // subdirectory path (if specified in URL)
}

// Manager handles all git operations.
type Manager struct{}

// NewManager creates a new git manager.
func NewManager() *Manager {
	return &Manager{}
}

// CheckGitVersion ensures git >= 2.25 is installed (required for sparse-checkout).
func (m *Manager) CheckGitVersion() error {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return fmt.Errorf("git is not installed or not in PATH: %w", err)
	}

	re := regexp.MustCompile(`git version (\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(string(out))
	if matches == nil {
		return fmt.Errorf("could not parse git version from: %s", string(out))
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	if major < 2 || (major == 2 && minor < 25) {
		return fmt.Errorf("git version must be >= 2.25, found %d.%d", major, minor)
	}
	return nil
}

// NormalizeURL parses a GitHub URL into its components.
// Supports formats:
//   - https://github.com/user/repo
//   - https://github.com/user/repo/tree/branch/path/to/skill
func (m *Manager) NormalizeURL(inputURL string) URLInfo {
	url := strings.TrimRight(inputURL, "/")

	// Strip query parameters and fragments
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}
	if idx := strings.Index(url, "#"); idx != -1 {
		url = url[:idx]
	}

	// Strip .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Match tree/branch/path format (with subdirectory)
	treePathRe := regexp.MustCompile(`^(https://github\.com/[^/]+/[^/]+)/tree/([^/]+)/(.+)$`)
	if matches := treePathRe.FindStringSubmatch(url); matches != nil {
		pathValue := strings.TrimRight(matches[3], "/")
		return URLInfo{
			URL:    matches[1] + ".git",
			Branch: matches[2],
			Path:   pathValue,
		}
	}

	// Match tree/branch format (root, no subdirectory)
	treeBranchRe := regexp.MustCompile(`^(https://github\.com/[^/]+/[^/]+)/tree/([^/]+)$`)
	if matches := treeBranchRe.FindStringSubmatch(url); matches != nil {
		return URLInfo{
			URL:    matches[1] + ".git",
			Branch: matches[2],
		}
	}

	// Plain repo URL
	return URLInfo{
		URL: url + ".git",
	}
}

// GetRemoteHead returns the commit hash pointed to by a remote ref.
func (m *Manager) GetRemoteHead(url string, branch string) (string, error) {
	if branch == "" {
		branch = "HEAD"
	}
	out, err := exec.Command("git", "ls-remote", url, branch).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote HEAD for %s %s: %w", url, branch, err)
	}
	re := regexp.MustCompile(`^([a-f0-9]+)\t`)
	matches := re.FindStringSubmatch(string(out))
	if matches == nil {
		return "", fmt.Errorf("could not parse remote HEAD for %s %s", url, branch)
	}
	return matches[1], nil
}

// CloneFull performs a full git clone.
func (m *Manager) CloneFull(url, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CloneSparse performs a sparse checkout of a specific subdirectory.
func (m *Manager) CloneSparse(url, dest, subPath, branch string) error {
	if branch == "" {
		branch = "main"
	}

	// Clone with blob filter and no checkout
	cmd := exec.Command("git", "clone", "--filter=blob:none", "--no-checkout", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sparse clone failed: %w", err)
	}

	// Init sparse checkout
	cmd = exec.Command("git", "sparse-checkout", "init", "--cone")
	cmd.Dir = dest
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sparse-checkout init failed: %w", err)
	}

	// Set sparse checkout path
	cmd = exec.Command("git", "sparse-checkout", "set", subPath)
	cmd.Dir = dest
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sparse-checkout set failed: %w", err)
	}

	// Checkout branch
	cmd = exec.Command("git", "checkout", branch)
	cmd.Dir = dest
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("checkout %s failed: %w", branch, err)
	}

	return nil
}

// Pull runs git pull in the given directory.
func (m *Manager) Pull(cwd string) error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Fetch runs git fetch origin in the given directory.
func (m *Manager) Fetch(cwd string) error {
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = cwd
	return cmd.Run()
}

// CheckRemoteSkillMd checks if SKILL.md exists at the given path in a remote repo.
func (m *Manager) CheckRemoteSkillMd(userRepo, branch, subPath string) bool {
	url := "https://github.com/" + userRepo + ".git"
	skillPath := "SKILL.md"
	if subPath != "" {
		skillPath = subPath + "/SKILL.md"
	}

	// Try git archive first
	refOut, err := exec.Command("git", "ls-remote", url, "refs/heads/"+branch).Output()
	if err != nil {
		return false
	}
	re := regexp.MustCompile(`^([a-f0-9]+)\t`)
	matches := re.FindStringSubmatch(string(refOut))
	if matches == nil {
		return false
	}
	commitHash := matches[1]

	// Try git archive (may not be supported by all hosts)
	if err := exec.Command("git", "archive", "--remote", url, commitHash, skillPath).Run(); err == nil {
		return true
	}

	// Fallback: shallow clone to temp dir
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("skm-check-%d", os.Getpid()))
	defer os.RemoveAll(tmpDir)

	checkPath := "."
	if subPath != "" {
		checkPath = subPath
	}

	cmd := exec.Command("git", "clone", "--depth=1", "--filter=blob:none", "--no-checkout", url, tmpDir)
	if err := cmd.Run(); err != nil {
		return false
	}

	cmd = exec.Command("git", "sparse-checkout", "init", "--cone")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		return false
	}

	cmd = exec.Command("git", "sparse-checkout", "set", checkPath)
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		return false
	}

	cmd = exec.Command("git", "checkout", branch)
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		return false
	}

	skillMdPath := filepath.Join(tmpDir, skillPath)
	_, err = os.Stat(skillMdPath)
	return err == nil
}

// GetDefaultBranch returns the default branch of a remote repo.
func (m *Manager) GetDefaultBranch(userRepo string) string {
	url := "https://github.com/" + userRepo + ".git"

	out, err := exec.Command("git", "ls-remote", "--symref", url, "HEAD").Output()
	if err == nil {
		re := regexp.MustCompile(`ref: refs/heads/([^\t\n]+)`)
		if matches := re.FindStringSubmatch(string(out)); matches != nil {
			return matches[1]
		}
	}
	return "main"
}

// GetLocalPathCommitID returns the latest commit hash for a path in a local repo.
func (m *Manager) GetLocalPathCommitID(repoDir, subPath string) (string, error) {
	out, err := exec.Command("git", "-C", repoDir, "log", "-1", "--format=%H", "--", subPath).Output()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return strings.TrimSpace(string(out)), nil
	}

	// Fallback to HEAD
	out, err = exec.Command("git", "-C", repoDir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD for %s: %w", repoDir, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetRemotePathCommitID returns the latest commit hash for a path on a remote branch.
// Should be called after Fetch.
func (m *Manager) GetRemotePathCommitID(repoDir, remoteBranch, subPath string) (string, error) {
	out, err := exec.Command("git", "-C", repoDir, "log", "-1", "--format=%H", remoteBranch, "--", subPath).Output()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return strings.TrimSpace(string(out)), nil
	}

	// Fallback to remote branch HEAD
	out, err = exec.Command("git", "-C", repoDir, "rev-parse", remoteBranch).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get %s HEAD for %s: %w", remoteBranch, repoDir, err)
	}
	return strings.TrimSpace(string(out)), nil
}
