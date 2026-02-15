package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/ArdentaCorp/agent-management/internal/config"
	"github.com/ArdentaCorp/agent-management/internal/project"
	"github.com/ArdentaCorp/agent-management/internal/skills"
	"github.com/ArdentaCorp/agent-management/internal/tui"
	"github.com/charmbracelet/huh"
)

// LinkToProject is the top-level "Link to project" flow.
// Detects tools, lets user pick one, then toggle skills.
func LinkToProject() {
	cm := config.NewManager()
	registry := skills.NewRegistry(cm)

	allSkills := registry.GetAllSkills()
	if len(allSkills) == 0 {
		fmt.Println(tui.RenderWarning("No skills in repository. Add skills first."))
		return
	}

	detector := project.NewDetector("")
	projects := detector.DetectAll()
	if len(projects) == 0 {
		fmt.Println(tui.RenderWarning("No AI tools detected in current directory."))
		fmt.Println(tui.MutedText.Render("  Supported: .cursor/ .claude/ .codex/ .copilot/ .gemini/"))
		return
	}

	// Pick tool (skip if only one)
	var selectedProjects []project.Info
	if len(projects) == 1 {
		selectedProjects = projects
		fmt.Println(tui.RenderInfo("Detected: " + projects[0].Type))
	} else {
		var opts []huh.Option[int]
		opts = append(opts, huh.NewOption("🔗 All detected tools", -1))
		for i, p := range projects {
			opts = append(opts, huh.NewOption(p.Type, i))
		}
		var idx int
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[int]().
				Title("Which tool?").
				Options(opts...).
				Value(&idx),
		))
		if err := form.Run(); err != nil {
			return
		}
		if idx == -1 {
			selectedProjects = projects
		} else {
			selectedProjects = []project.Info{projects[idx]}
		}
	}

	// For each selected tool, run the link flow
	for _, selectedProject := range selectedProjects {
		fmt.Print(tui.RenderSection(selectedProject.Type + " Skills"))
		fmt.Println(tui.MutedText.Render("  " + selectedProject.SkillDir))

		os.MkdirAll(selectedProject.SkillDir, 0755)

		// Check broken symlinks
		brokenLinks := findBrokenLinks(selectedProject.SkillDir)
		if len(brokenLinks) > 0 {
			fmt.Println(tui.RenderWarning(fmt.Sprintf("Found %d broken symlink(s)", len(brokenLinks))))
			var cleanup bool
			huh.NewForm(huh.NewGroup(
				huh.NewConfirm().
					Title("Remove broken symlinks?").
					Value(&cleanup),
			)).Run()
			if cleanup {
				for _, link := range brokenLinks {
					os.Remove(filepath.Join(selectedProject.SkillDir, link))
					fmt.Println(tui.RenderSuccess("Removed " + link))
				}
			}
		}

		// Show other skills
		otherSkills := findOtherSkills(selectedProject.SkillDir)
		if len(otherSkills) > 0 {
			sort.Strings(otherSkills)
			fmt.Println(tui.MutedText.Render("\n  Other skills (not managed by agm):"))
			for _, name := range otherSkills {
				fmt.Println(tui.MutedText.Render("    • " + name))
			}
			fmt.Println()
		}

		// Build multiselect
		linkedSkills := getLinkedSkills(allSkills, cm, selectedProject.SkillDir)

		var skillOpts []huh.Option[string]
		for _, skill := range allSkills {
			label := skill.ID
			if linkedSkills[skill.ID] {
				label += " " + tui.SuccessText.Render("✓")
			}
			opt := huh.NewOption(label, skill.ID)
			if linkedSkills[skill.ID] {
				opt = opt.Selected(true)
			}
			skillOpts = append(skillOpts, opt)
		}

		var selected []string
		toggleForm := huh.NewForm(huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Toggle skills").
				Description("Space to toggle, Enter to apply").
				Options(skillOpts...).
				Value(&selected),
		))
		if err := toggleForm.Run(); err != nil {
			return
		}

		selectedSet := make(map[string]bool)
		for _, id := range selected {
			selectedSet[id] = true
		}

		changes := 0
		p := selectedProject
		for _, skill := range allSkills {
			isLinked := linkedSkills[skill.ID]
			shouldBeLinked := selectedSet[skill.ID]

			if !isLinked && shouldBeLinked {
				linkSkillToProject(skill.ID, &p)
				changes++
			} else if isLinked && !shouldBeLinked {
				unlinkSkillFromProject(skill.ID, &p)
				changes++
			}
		}

		if changes == 0 {
			fmt.Println(tui.MutedText.Render("\nNo changes."))
		} else {
			fmt.Printf("\n%s\n", tui.RenderSuccess(fmt.Sprintf("%d change(s) applied to %s", changes, selectedProject.Type)))
		}
	}
}

// linkSkillToProject creates a symlink from the global repo to the project.
func linkSkillToProject(skillID string, projectInfo *project.Info) {
	cm := config.NewManager()
	registry := skills.NewRegistry(cm)

	skill := registry.GetSkill(skillID)
	if skill == nil {
		fmt.Println(tui.RenderError("Skill " + skillID + " not found."))
		return
	}

	os.MkdirAll(projectInfo.SkillDir, 0755)

	linkName := cm.GetLinkName(skill.ID)
	linkPath := filepath.Join(projectInfo.SkillDir, linkName)
	repoPath := cm.GetRepoPath(skill.ID)

	targetPath := repoPath
	if skill.Path != "" {
		targetPath = filepath.Join(repoPath, skill.Path)
	}

	if _, err := os.Lstat(linkPath); err == nil {
		return // already linked
	}

	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "mklink", "/J", linkPath, targetPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Println(tui.RenderError(fmt.Sprintf("Failed to link %s: %v\n%s", skill.ID, err, output)))
			return
		}
	} else {
		if err := os.Symlink(targetPath, linkPath); err != nil {
			fmt.Println(tui.RenderError("Failed to link " + skill.ID + ": " + err.Error()))
			return
		}
	}

	fmt.Println(tui.RenderSuccess("Linked " + skill.ID))
}

// unlinkSkillFromProject removes a symlink.
func unlinkSkillFromProject(skillID string, projectInfo *project.Info) {
	cm := config.NewManager()
	linkName := cm.GetLinkName(skillID)
	linkPath := filepath.Join(projectInfo.SkillDir, linkName)

	if _, err := os.Lstat(linkPath); os.IsNotExist(err) {
		return
	}

	if err := os.Remove(linkPath); err != nil {
		fmt.Println(tui.RenderError("Failed to unlink " + skillID + ": " + err.Error()))
		return
	}
	fmt.Println(tui.RenderSuccess("Unlinked " + skillID))
}

// --- helpers ---

func getLinkedSkills(allSkills []skills.Skill, cm *config.Manager, skillDir string) map[string]bool {
	linked := make(map[string]bool)
	for _, skill := range allSkills {
		linkName := cm.GetLinkName(skill.ID)
		linkPath := filepath.Join(skillDir, linkName)
		if _, err := os.Lstat(linkPath); err == nil {
			linked[skill.ID] = true
		}
	}
	return linked
}

func findBrokenLinks(skillDir string) []string {
	var broken []string
	entries, err := os.ReadDir(skillDir)
	if err != nil {
		return broken
	}
	for _, entry := range entries {
		linkPath := filepath.Join(skillDir, entry.Name())
		info, err := os.Lstat(linkPath)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(linkPath)
			if err != nil {
				broken = append(broken, entry.Name())
				continue
			}
			if _, err := os.Stat(target); os.IsNotExist(err) {
				broken = append(broken, entry.Name())
			}
		}
	}
	return broken
}

func findOtherSkills(skillDir string) []string {
	var others []string
	entries, err := os.ReadDir(skillDir)
	if err != nil {
		return others
	}
	for _, entry := range entries {
		entryPath := filepath.Join(skillDir, entry.Name())
		info, err := os.Lstat(entryPath)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if info.IsDir() {
			if _, err := os.Stat(filepath.Join(entryPath, "SKILL.md")); err == nil {
				others = append(others, entry.Name())
			}
		}
	}
	return others
}
