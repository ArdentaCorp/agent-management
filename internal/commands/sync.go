package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArdentaCorp/agent-management/internal/config"
	"github.com/ArdentaCorp/agent-management/internal/git"
	"github.com/ArdentaCorp/agent-management/internal/skills"
	"github.com/ArdentaCorp/agent-management/internal/tui"
	"github.com/charmbracelet/huh"
)

// SyncSkills syncs all skills from the configured registry repo.
// If interactive is true, prompts for the URL when not configured.
// If interactive is false (--sync flag), fails if no registry is configured.
func SyncSkills(interactive bool) {
	cm := config.NewManager()
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
		if err := gitMgr.CloneFull(cloneURL, registryDir); err != nil {
			fmt.Println(tui.RenderError("Failed to clone registry: " + err.Error()))
			return
		}
	} else {
		// Already cloned — pull latest
		fmt.Println(tui.RenderInfo("Pulling latest changes..."))
		if err := gitMgr.Pull(registryDir); err != nil {
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

	for _, skillDir := range foundSkills {
		skillName := filepath.Base(skillDir)
		id := "registry:" + skillName
		destPath := cm.GetRepoPath(id)

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

	// Remove skills that are no longer in the registry
	allSkills := registry.GetAllSkills()
	foundSet := make(map[string]bool)
	for _, skillDir := range foundSkills {
		foundSet["registry:"+filepath.Base(skillDir)] = true
	}

	removed := 0
	for _, skill := range allSkills {
		if skill.Type == "registry" && !foundSet[skill.ID] {
			os.RemoveAll(cm.GetRepoPath(skill.ID))
			registry.RemoveSkill(skill.ID)
			fmt.Println(tui.RenderWarning("  - " + cm.GetLinkName(skill.ID) + " (removed from registry)"))
			removed++
		}
	}
	if removed > 0 {
		fmt.Println(tui.RenderInfo(fmt.Sprintf("%d skill(s) removed (no longer in registry)", removed)))
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
