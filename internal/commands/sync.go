package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArdentaCorp/agent-management/internal/config"
	"github.com/ArdentaCorp/agent-management/internal/git"
	"github.com/ArdentaCorp/agent-management/internal/project"
	"github.com/ArdentaCorp/agent-management/internal/skills"
	"github.com/ArdentaCorp/agent-management/internal/tui"
	"github.com/charmbracelet/huh"
)

// SyncSkills syncs all skills from the configured registry repo.
// If interactive is true, prompts for the URL when not configured.
// If interactive is false (--sync flag), fails if no registry is configured.
func SyncSkills(interactive bool) {
	cm, err := config.NewManager()
	if err != nil {
		fmt.Println(tui.RenderError("Failed to initialize config: " + err.Error()))
		return
	}
	gitMgr := git.NewManager()

	if err := gitMgr.CheckGitVersion(); err != nil {
		fmt.Println(tui.RenderError(err.Error()))
		return
	}

	registryURL := cm.GetRegistry()

	if registryURL == "" {
		if !interactive {
			fmt.Println(tui.RenderError("No registry configured. Run agm and use 'Sync skills' to set one up."))
			return
		}

		// First time — ask for the URL
		var inputURL string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Registry URL").
					Description("GitHub repo containing your team's skills").
					Placeholder("https://github.com/org/skills").
					Value(&inputURL),
			),
		)
		if err := form.Run(); err != nil || strings.TrimSpace(inputURL) == "" {
			return
		}

		registryURL = strings.TrimSpace(inputURL)
		if err := cm.SetRegistry(registryURL); err != nil {
			fmt.Println(tui.RenderError("Failed to save registry URL: " + err.Error()))
			return
		}
		fmt.Println(tui.RenderSuccess("Registry saved: " + registryURL))
	}

	registryDir := cm.GetRegistryDir()

	// Parse the URL — supports GitHub browse URLs like .../tree/main/skills
	info := gitMgr.NormalizeURL(registryURL)
	cloneURL := info.URL

	// Clone or pull
	if _, err := os.Stat(filepath.Join(registryDir, ".git")); os.IsNotExist(err) {
		// First time — clone
		fmt.Println(tui.RenderInfo("Cloning registry..."))
		os.MkdirAll(filepath.Dir(registryDir), 0755)
		if err := gitMgr.CloneFullQuiet(cloneURL, registryDir); err != nil {
			fmt.Println(tui.RenderError("Failed to clone registry: " + err.Error()))
			return
		}
	} else {
		// Already cloned — pull latest
		fmt.Println(tui.RenderInfo("Pulling latest changes..."))
		if err := gitMgr.PullQuiet(registryDir); err != nil {
			fmt.Println(tui.RenderError("Failed to pull: " + err.Error()))
			return
		}
	}

	// If the URL pointed to a subfolder (e.g. /tree/main/skills), scan that subfolder
	scanRoot := registryDir
	if info.Path != "" {
		scanRoot = filepath.Join(registryDir, info.Path)
	}

	// Scan for skills (directories containing SKILL.md)
	foundSkills := scanForSkills(scanRoot)
	if len(foundSkills) == 0 {
		fmt.Println(tui.RenderWarning("No skills found in registry (no SKILL.md files)."))
		return
	}

	// Sync each skill into the repo
	registry := skills.NewRegistry(cm)
	added := 0
	updated := 0
	unchanged := 0
	replaced := 0
	replacedLinks := 0
	detectedProjects := project.NewDetector("").DetectAll()

	for _, skillDir := range foundSkills {
		skillName := filepath.Base(skillDir)
		id := "registry:" + skillName
		destPath := cm.GetRepoPath(id)

		removedSources, removedLinks := removeSkillsWithLinkName(cm, registry, skillName, id, detectedProjects)
		if removedSources > 0 {
			replaced += removedSources
			replacedLinks += removedLinks
			fmt.Println(tui.RenderWarning(fmt.Sprintf("  ~ %s: replaced %d existing source(s) with registry", skillName, removedSources)))
		}

		existing := registry.GetSkill(id)

		// Copy skill directory to repo
		if err := os.RemoveAll(destPath); err != nil && !os.IsNotExist(err) {
			fmt.Println(tui.RenderError("Failed to clean " + skillName + ": " + err.Error()))
			continue
		}

		if err := copyDir(skillDir, destPath); err != nil {
			fmt.Println(tui.RenderError("Failed to copy " + skillName + ": " + err.Error()))
			continue
		}

		// Get commit for this skill's path (relative to repo root, not scan root)
		relPath, _ := filepath.Rel(registryDir, skillDir)
		commitID, _ := gitMgr.GetLocalPathCommitID(registryDir, filepath.ToSlash(relPath))

		registry.AddSkill(id, "registry", commitID, "")

		if existing == nil {
			fmt.Println(tui.RenderSuccess("  + " + skillName + " (new)"))
			added++
		} else if existing.CommitID != commitID {
			fmt.Println(tui.RenderSuccess("  ↑ " + skillName + " (updated)"))
			updated++
		} else {
			unchanged++
		}
	}

	fmt.Println()
	summary := fmt.Sprintf("Sync complete: %d new, %d updated, %d unchanged", added, updated, unchanged)
	fmt.Println(tui.RenderSuccess(summary))
	if replaced > 0 {
		fmt.Println(tui.RenderInfo(fmt.Sprintf("%d duplicate source skill(s) replaced by registry", replaced)))
		if replacedLinks > 0 {
			fmt.Println(tui.RenderInfo(fmt.Sprintf("%d linked skill entry(s) removed for replaced sources", replacedLinks)))
		}
	}

	// Remove skills that are no longer in the registry
	allSkills := registry.GetAllSkills()
	foundSet := make(map[string]bool)
	for _, skillDir := range foundSkills {
		foundSet["registry:"+filepath.Base(skillDir)] = true
	}

	removed := 0
	linkCleanup := 0
	for _, skill := range allSkills {
		if skill.Type == "registry" && !foundSet[skill.ID] {
			os.RemoveAll(cm.GetRepoPath(skill.ID))
			for _, p := range detectedProjects {
				if removeSkillLinkIfPresent(cm, skill.ID, p) {
					linkCleanup++
				}
			}
			registry.RemoveSkill(skill.ID)
			fmt.Println(tui.RenderWarning("  - " + cm.GetLinkName(skill.ID) + " (removed from registry)"))
			removed++
		}
	}
	if removed > 0 {
		fmt.Println(tui.RenderInfo(fmt.Sprintf("%d skill(s) removed (no longer in registry)", removed)))
		if linkCleanup > 0 {
			fmt.Println(tui.RenderInfo(fmt.Sprintf("%d linked skill entry(s) removed from detected project tools", linkCleanup)))
		}
	}
}

// scanForSkills walks the registry directory and returns paths of directories containing SKILL.md.
// Only scans one level deep (direct children of the root).
func scanForSkills(root string) []string {
	var results []string

	entries, err := os.ReadDir(root)
	if err != nil {
		return results
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		skillMd := filepath.Join(root, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillMd); err == nil {
			results = append(results, filepath.Join(root, entry.Name()))
		}
	}

	return results
}

func removeSkillLinkIfPresent(cm *config.Manager, skillID string, projectInfo project.Info) bool {
	linkPath := filepath.Join(projectInfo.SkillDir, cm.GetLinkName(skillID))
	if _, err := os.Lstat(linkPath); err != nil {
		return false
	}
	if err := os.Remove(linkPath); err != nil {
		return false
	}
	return true
}

func removeSkillsWithLinkName(cm *config.Manager, registry *skills.Registry, linkName, keepID string, detectedProjects []project.Info) (int, int) {
	removedSources := 0
	removedLinks := 0
	for _, skill := range registry.GetAllSkills() {
		if skill.ID == keepID {
			continue
		}
		if cm.GetLinkName(skill.ID) != linkName {
			continue
		}
		_ = os.RemoveAll(cm.GetRepoPath(skill.ID))
		for _, p := range detectedProjects {
			if removeSkillLinkIfPresent(cm, skill.ID, p) {
				removedLinks++
			}
		}
		registry.RemoveSkill(skill.ID)
		removedSources++
	}
	return removedSources, removedLinks
}
