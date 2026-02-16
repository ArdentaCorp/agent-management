package skills

import (
	"path/filepath"
	"testing"
)

func TestRegistryCRUD(t *testing.T) {
	t.Parallel()

	r := &Registry{versionsFile: filepath.Join(t.TempDir(), "skills.json")}

	r.AddSkill("github:user/repo/a-skill", "github", "abc123", "skills/a-skill")
	r.AddSkill("local:local-skill", "local", "", "")

	s := r.GetSkill("github:user/repo/a-skill")
	if s == nil {
		t.Fatal("expected skill to exist")
	}
	if s.Type != "github" {
		t.Fatalf("unexpected type: %s", s.Type)
	}
	if s.Path != "skills/a-skill" {
		t.Fatalf("unexpected path: %s", s.Path)
	}
	if s.CommitID != "abc123" {
		t.Fatalf("unexpected commit: %s", s.CommitID)
	}

	r.UpdateSkillVersion("github:user/repo/a-skill", "def456")
	s = r.GetSkill("github:user/repo/a-skill")
	if s == nil || s.CommitID != "def456" {
		t.Fatalf("commit not updated, got %+v", s)
	}

	all := r.GetAllSkills()
	if len(all) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(all))
	}
	if all[0].ID != "github:user/repo/a-skill" || all[1].ID != "local:local-skill" {
		t.Fatalf("skills are not sorted by ID: %+v", all)
	}

	r.RemoveSkill("local:local-skill")
	if got := r.GetSkill("local:local-skill"); got != nil {
		t.Fatalf("expected local skill to be removed, got %+v", got)
	}
}
