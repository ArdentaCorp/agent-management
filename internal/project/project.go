package project

import (
	"os"
	"path/filepath"

	"github.com/ArdentaCorp/agent-management/internal/config"
)

// Info holds detected project information.
type Info struct {
	Type     string
	Root     string
	SkillDir string
}

// Detector auto-detects AI tool project types in a directory.
type Detector struct {
	cwd     string
	aiTools []config.AIToolConfig
}

// DefaultAITools is the built-in list of supported AI tool configurations.
var DefaultAITools = []config.AIToolConfig{
	{Type: "antigravity", SkillDirs: []string{".gemini/antigravity/global_skills/skills", ".agent/skills"}},
	{Type: "github", SkillDirs: []string{".copilot/skills", ".github/skills"}},
	{Type: "cursor", SkillDirs: []string{".cursor/skills"}},
	{Type: "claude", SkillDirs: []string{".claude/skills"}},
	{Type: "codex", SkillDirs: []string{".codex/skills", ".agents/skills"}},
}

// NewDetector creates a new project detector for the given directory.
// If cwd is empty, the current working directory is used.
func NewDetector(cwd string) *Detector {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	aiTools := DefaultAITools
	if cm, err := config.NewManager(); err == nil {
		if cfgTools := cm.GetAITools(); cfgTools != nil {
			aiTools = cfgTools
		}
	}

	return &Detector{
		cwd:     cwd,
		aiTools: aiTools,
	}
}

// DetectAll returns all detected AI project types in the directory.
// A tool is detected if any of its skillDir parent directories exist.
func (d *Detector) DetectAll() []Info {
	var projects []Info

	for _, tool := range d.aiTools {
		for _, skillDir := range tool.SkillDirs {
			fullSkillDir := filepath.Join(d.cwd, skillDir)
			parentDir := filepath.Dir(fullSkillDir)
			if _, err := os.Stat(parentDir); err == nil {
				projects = append(projects, Info{
					Type:     tool.Type,
					Root:     d.cwd,
					SkillDir: fullSkillDir,
				})
				break // found one for this tool, move on
			}
		}
	}

	return projects
}

// Detect returns the first detected AI project type (backward compat).
func (d *Detector) Detect() Info {
	projects := d.DetectAll()
	if len(projects) > 0 {
		return projects[0]
	}
	return Info{
		Type: "unknown",
		Root: d.cwd,
	}
}
