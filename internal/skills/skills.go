package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/ArdentaCorp/agent-management/internal/config"
)

// Skill represents a registered skill with its metadata.
type Skill struct {
	ID       string `json:"id"`
	CommitID string `json:"commitId,omitempty"`
	Type     string `json:"type"`
	Path     string `json:"path,omitempty"`
}

// storedSkill is the JSON storage format (without ID, since ID is the map key).
type storedSkill struct {
	CommitID string `json:"commitId,omitempty"`
	Type     string `json:"type"`
	Path     string `json:"path,omitempty"`
}

// Registry manages the skills.json registry file.
type Registry struct {
	versionsFile string
}

// NewRegistry creates a new skill registry.
func NewRegistry(cm *config.Manager) *Registry {
	return &Registry{
		versionsFile: filepath.Join(cm.GetRepoDir(), "skills.json"),
	}
}

func (r *Registry) load() map[string]storedSkill {
	data, err := os.ReadFile(r.versionsFile)
	if err != nil {
		return make(map[string]storedSkill)
	}
	var skills map[string]storedSkill
	if err := json.Unmarshal(data, &skills); err != nil {
		return make(map[string]storedSkill)
	}
	return skills
}

func (r *Registry) save(skills map[string]storedSkill) {
	data, _ := json.MarshalIndent(skills, "", "  ")
	os.WriteFile(r.versionsFile, data, 0644)
}

// AddSkill registers a new skill.
func (r *Registry) AddSkill(id, skillType, commitID, skillPath string) {
	skills := r.load()
	s := storedSkill{Type: skillType}
	if commitID != "" {
		s.CommitID = commitID
	}
	if skillPath != "" {
		s.Path = skillPath
	}
	skills[id] = s
	r.save(skills)
}

// RemoveSkill removes a skill from the registry.
func (r *Registry) RemoveSkill(id string) {
	skills := r.load()
	delete(skills, id)
	r.save(skills)
}

// GetSkill returns a skill by ID, or nil if not found.
func (r *Registry) GetSkill(id string) *Skill {
	skills := r.load()
	stored, ok := skills[id]
	if !ok {
		return nil
	}
	return &Skill{
		ID:       id,
		CommitID: stored.CommitID,
		Type:     stored.Type,
		Path:     stored.Path,
	}
}

// UpdateSkillVersion updates the commit ID for a skill.
func (r *Registry) UpdateSkillVersion(id, newCommitID string) {
	skills := r.load()
	if s, ok := skills[id]; ok {
		s.CommitID = newCommitID
		skills[id] = s
		r.save(skills)
	}
}

// GetAllSkills returns all registered skills, sorted by ID.
func (r *Registry) GetAllSkills() []Skill {
	skills := r.load()
	result := make([]Skill, 0, len(skills))
	for id, stored := range skills {
		result = append(result, Skill{
			ID:       id,
			CommitID: stored.CommitID,
			Type:     stored.Type,
			Path:     stored.Path,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result
}
