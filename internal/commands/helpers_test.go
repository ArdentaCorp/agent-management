package commands

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ArdentaCorp/agent-management/internal/config"
	"github.com/ArdentaCorp/agent-management/internal/project"
	"github.com/ArdentaCorp/agent-management/internal/skills"
)

func TestScanForSkillsOneLevel(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "skill-a"))
	mustWriteFile(t, filepath.Join(root, "skill-a", "SKILL.md"), "a")

	mustMkdirAll(t, filepath.Join(root, "skill-b", "nested"))
	mustWriteFile(t, filepath.Join(root, "skill-b", "nested", "SKILL.md"), "nested")

	mustMkdirAll(t, filepath.Join(root, ".hidden-skill"))
	mustWriteFile(t, filepath.Join(root, ".hidden-skill", "SKILL.md"), "hidden")

	found := scanForSkills(root)
	for i := range found {
		found[i] = filepath.Base(found[i])
	}
	slices.Sort(found)

	want := []string{"skill-a"}
	if !slices.Equal(found, want) {
		t.Fatalf("scanForSkills() = %v, want %v", found, want)
	}
}

func TestCopyDirCopiesNestedTree(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")

	mustMkdirAll(t, filepath.Join(src, "nested"))
	mustWriteFile(t, filepath.Join(src, "SKILL.md"), "root")
	mustWriteFile(t, filepath.Join(src, "nested", "notes.txt"), "nested-content")

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir() returned error: %v", err)
	}

	assertFileContent(t, filepath.Join(dst, "SKILL.md"), "root")
	assertFileContent(t, filepath.Join(dst, "nested", "notes.txt"), "nested-content")
}

func TestRemoveSkillLinkIfPresent(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	cm := &config.Manager{}
	skillID := "registry:team-skill"
	linkName := cm.GetLinkName(skillID)
	linkPath := filepath.Join(projectDir, linkName)

	mustWriteFile(t, linkPath, "placeholder")

	removed := removeSkillLinkIfPresent(cm, skillID, project.Info{SkillDir: projectDir})
	if !removed {
		t.Fatal("expected link cleanup to remove existing entry")
	}
	if _, err := os.Stat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("expected cleaned path to be removed, stat err: %v", err)
	}

	removed = removeSkillLinkIfPresent(cm, skillID, project.Info{SkillDir: projectDir})
	if removed {
		t.Fatal("expected false when no linked entry exists")
	}
}

func TestRemoveSkillsWithLinkName(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	cm, err := config.NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	registry := skills.NewRegistry(cm)

	githubID := "github:org/repo/skills/create-migration"
	localID := "local:create-migration"
	keepID := "registry:create-migration"

	registry.AddSkill(githubID, "github", "abc123", "skills/create-migration")
	registry.AddSkill(localID, "local", "", "")
	registry.AddSkill(keepID, "registry", "def456", "")

	mustMkdirAll(t, cm.GetRepoPath(githubID))
	mustMkdirAll(t, cm.GetRepoPath(localID))
	mustMkdirAll(t, cm.GetRepoPath(keepID))

	projectDir := t.TempDir()
	linkPath := filepath.Join(projectDir, cm.GetLinkName(githubID))
	mustWriteFile(t, linkPath, "linked")

	removedSources, removedLinks := removeSkillsWithLinkName(
		cm,
		registry,
		"create-migration",
		keepID,
		[]project.Info{{SkillDir: projectDir}},
	)

	if removedSources != 2 {
		t.Fatalf("removedSources = %d, want 2", removedSources)
	}
	if removedLinks != 1 {
		t.Fatalf("removedLinks = %d, want 1", removedLinks)
	}
	if registry.GetSkill(githubID) != nil {
		t.Fatal("expected github duplicate to be removed from registry")
	}
	if registry.GetSkill(localID) != nil {
		t.Fatal("expected local duplicate to be removed from registry")
	}
	if registry.GetSkill(keepID) == nil {
		t.Fatal("expected keepID skill to remain")
	}
	if _, err := os.Stat(cm.GetRepoPath(githubID)); !os.IsNotExist(err) {
		t.Fatalf("expected github duplicate repo to be removed, got err: %v", err)
	}
	if _, err := os.Stat(cm.GetRepoPath(localID)); !os.IsNotExist(err) {
		t.Fatalf("expected local duplicate repo to be removed, got err: %v", err)
	}
	if _, err := os.Stat(cm.GetRepoPath(keepID)); err != nil {
		t.Fatalf("expected keepID repo to remain, got err: %v", err)
	}
	if _, err := os.Stat(linkPath); !os.IsNotExist(err) {
		t.Fatalf("expected duplicate linked entry to be removed, got err: %v", err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) failed: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) failed: %v", path, err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) failed: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("file %s content = %q, want %q", path, string(data), want)
	}
}
