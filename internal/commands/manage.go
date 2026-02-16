package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/ArdentaCorp/agent-management/internal/config"
	"github.com/ArdentaCorp/agent-management/internal/git"
	"github.com/ArdentaCorp/agent-management/internal/skills"
	"github.com/ArdentaCorp/agent-management/internal/tui"
	"github.com/charmbracelet/huh"
)

// ManageSkills is the top-level "Manage skills" flow.
// Shows all installed skills, lets user update/delete.
func ManageSkills() {
	cm, err := config.NewManager()
	if err != nil {
		fmt.Println(tui.RenderError("Failed to initialize config: " + err.Error()))
		return
	}
	registry := skills.NewRegistry(cm)

	for {
		allSkills := registry.GetAllSkills()

		if len(allSkills) == 0 {
			fmt.Println(tui.RenderWarning("No skills installed. Use 'Add skills' first."))
			return
		}

		fmt.Print(tui.RenderSection("Manage Skills"))

		var opts []huh.Option[string]
		for _, s := range allSkills {
			label := s.ID
			if s.Type == "local" {
				label += " " + tui.MutedText.Render("(local)")
			} else if s.CommitID != "" {
				label += " " + tui.MutedText.Render("("+s.CommitID[:min(7, len(s.CommitID))]+")")
			}
			opts = append(opts, huh.NewOption(label, s.ID))
		}
		opts = append(opts, huh.NewOption("← Back", "back"))

		var selected string
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a skill").
				Options(opts...).
				Value(&selected),
		))
		if err := form.Run(); err != nil || selected == "back" {
			return
		}

		skill := registry.GetSkill(selected)
		if skill == nil {
			continue
		}

		manageOneSkill(*skill)
	}
}

// manageOneSkill shows actions for a single skill.
func manageOneSkill(skill skills.Skill) {
	fmt.Print(tui.RenderSection(skill.ID))

	var opts []huh.Option[string]

	var update *updateInfo
	if skill.Type == "github" {
		update = checkForUpdate(skill)
		if update != nil {
			label := fmt.Sprintf("⬆️  Update (%s → %s)",
				truncate(skill.CommitID, 7),
				truncate(update.remoteHead, 7))
			opts = append(opts, huh.NewOption(label, "update"))
		} else {
			fmt.Println(tui.SuccessText.Render("  Up to date"))
		}
	} else {
		fmt.Println(tui.MutedText.Render("  Local — no remote updates"))
	}

	opts = append(opts,
		huh.NewOption("🗑️  Delete", "delete"),
		huh.NewOption("← Back", "back"),
	)

	var action string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Action").
			Options(opts...).
			Value(&action),
	)).Run(); err != nil {
		return
	}

	switch action {
	case "update":
		if update != nil {
			doUpdate(skill, *update)
		}
	case "delete":
		doDelete(skill.ID)
	}
}

type updateInfo struct {
	remoteHead string
	branch     string
}

func checkForUpdate(skill skills.Skill) *updateInfo {
	cm, err := config.NewManager()
	if err != nil {
		return nil
	}
	gitMgr := git.NewManager()

	parts := strings.SplitN(skill.ID, ":", 2)
	if len(parts) < 2 {
		return nil
	}
	repoPath := parts[1]

	userRepo := repoPath
	if skill.Path != "" && strings.HasSuffix(repoPath, skill.Path) {
		userRepo = repoPath[:len(repoPath)-len(skill.Path)-1]
	}

	localRepoDir := cm.GetRepoPath(skill.ID)

	subPath := "."
	if skill.Path != "" {
		subPath = skill.Path
	}
	localCommit, err := gitMgr.GetLocalPathCommitID(localRepoDir, subPath)
	if err != nil {
		return nil
	}

	fmt.Println(tui.RenderInfo("Checking for updates..."))
	if err := gitMgr.Fetch(localRepoDir); err != nil {
		return nil
	}

	branch := gitMgr.GetDefaultBranch(userRepo)
	remoteHead, err := gitMgr.GetRemotePathCommitID(localRepoDir, "origin/"+branch, subPath)
	if err != nil {
		return nil
	}

	if remoteHead != "" && remoteHead != localCommit {
		return &updateInfo{remoteHead: remoteHead, branch: branch}
	}
	return nil
}

func doUpdate(skill skills.Skill, info updateInfo) {
	cm, err := config.NewManager()
	if err != nil {
		fmt.Println(tui.RenderError("Failed to initialize config: " + err.Error()))
		return
	}
	registry := skills.NewRegistry(cm)
	gitMgr := git.NewManager()

	fmt.Println(tui.RenderInfo("Updating " + skill.ID + "..."))
	destPath := cm.GetRepoPath(skill.ID)

	if err := gitMgr.PullQuiet(destPath); err != nil {
		fmt.Println(tui.RenderError("Failed: " + err.Error()))
		return
	}

	registry.UpdateSkillVersion(skill.ID, info.remoteHead)
	fmt.Println(tui.RenderSuccess("Updated " + skill.ID))
}

func doDelete(id string) {
	cm, err := config.NewManager()
	if err != nil {
		fmt.Println(tui.RenderError("Failed to initialize config: " + err.Error()))
		return
	}
	registry := skills.NewRegistry(cm)

	var confirm bool
	if err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title(fmt.Sprintf("Delete %s? This cannot be undone.", id)).
			Affirmative("Yes, delete").
			Negative("Cancel").
			Value(&confirm),
	)).Run(); err != nil {
		fmt.Println(tui.MutedText.Render("Cancelled."))
		return
	}

	if !confirm {
		fmt.Println(tui.MutedText.Render("Cancelled."))
		return
	}

	os.RemoveAll(cm.GetRepoPath(id))
	registry.RemoveSkill(id)
	fmt.Println(tui.RenderSuccess("Deleted " + id))
}

// --- helpers ---

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
