package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ArdentaCorp/agent-management/internal/config"
	"github.com/ArdentaCorp/agent-management/internal/git"
	"github.com/ArdentaCorp/agent-management/internal/project"
	"github.com/ArdentaCorp/agent-management/internal/skills"
	"github.com/ArdentaCorp/agent-management/internal/tui"
	"github.com/charmbracelet/huh"
)

// AddSkills is the top-level "Add skills" flow.
// After adding, it offers to link to a detected project immediately.
func AddSkills() {
	cm, err := config.NewManager()
	if err != nil {
		fmt.Println(tui.RenderError("Failed to initialize config: " + err.Error()))
		return
	}
	registryURL := cm.GetRegistry()

	var opts []huh.Option[string]
	if registryURL != "" {
		opts = append(opts, huh.NewOption("🔄 Sync from registry", "sync"))
	} else {
		opts = append(opts, huh.NewOption("🔄 Set up registry", "sync"))
	}
	opts = append(opts,
		huh.NewOption("🌐 GitHub Repository", "github"),
		huh.NewOption("📁 Local Folder", "folder"),
		huh.NewOption("← Cancel", "cancel"),
	)

	var skillType string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Where are the skills?").
				Options(opts...).
				Value(&skillType),
		),
	)
	if err := form.Run(); err != nil {
		return
	}

	var addedIDs []string

	switch skillType {
	case "sync":
		SyncSkills(true)
		return
	case "github":
		addedIDs = addGitHubSkill()
	case "folder":
		addedIDs = addSkillsFolder()
	default:
		return
	}

	if len(addedIDs) == 0 {
		return
	}

	// Wizard: offer to link to project immediately
	offerLinkAfterAdd(addedIDs)
}

// offerLinkAfterAdd asks if the user wants to link newly added skills to a project.
func offerLinkAfterAdd(addedIDs []string) {
	detector := project.NewDetector("")
	projects := detector.DetectAll()
	if len(projects) == 0 {
		return
	}

	var wantLink bool
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Link these skills to a project now?").
				Affirmative("Yes").
				Negative("Not now").
				Value(&wantLink),
		),
	).Run(); err != nil {
		return
	}

	if !wantLink {
		return
	}

	// If only one tool detected, link directly
	if len(projects) == 1 {
		p := projects[0]
		for _, id := range addedIDs {
			linkSkillToProject(id, &p)
		}
		return
	}

	// Multiple tools — offer "All" or pick one
	var opts []huh.Option[int]
	opts = append(opts, huh.NewOption("🔗 All detected tools", -1))
	for i, p := range projects {
		opts = append(opts, huh.NewOption(p.Type, i))
	}
	var idx int
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Which tool?").
				Options(opts...).
				Value(&idx),
		),
	).Run(); err != nil {
		return
	}

	if idx == -1 {
		// Link to all detected tools
		for _, p := range projects {
			fmt.Println(tui.RenderInfo("Linking to " + p.Type + "..."))
			for _, id := range addedIDs {
				linkSkillToProject(id, &p)
			}
		}
	} else {
		if idx < 0 || idx >= len(projects) {
			return
		}
		p := projects[idx]
		for _, id := range addedIDs {
			linkSkillToProject(id, &p)
		}
	}
}

// addGitHubSkill adds a skill from a GitHub URL. Returns added skill IDs.
func addGitHubSkill() []string {
	cm, err := config.NewManager()
	if err != nil {
		fmt.Println(tui.RenderError("Failed to initialize config: " + err.Error()))
		return nil
	}
	registry := skills.NewRegistry(cm)
	gitMgr := git.NewManager()

	var repoURL string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("GitHub URL").
				Description("Repo or subdirectory URL").
				Placeholder("https://github.com/user/repo/tree/main/skills/...").
				Value(&repoURL),
		),
	)
	if err := form.Run(); err != nil || repoURL == "" {
		return nil
	}

	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return nil
	}

	if err := gitMgr.CheckGitVersion(); err != nil {
		fmt.Println(tui.RenderError(err.Error()))
		return nil
	}

	gitInfo := gitMgr.NormalizeURL(repoURL)

	re := regexp.MustCompile(`github\.com/([^/]+/[^/]+?)(\.git)?$`)
	matches := re.FindStringSubmatch(gitInfo.URL)
	if matches == nil {
		fmt.Println(tui.RenderError("Only GitHub URLs are supported."))
		return nil
	}
	userRepo := strings.TrimSuffix(matches[1], ".git")

	branch := gitInfo.Branch
	if branch == "" {
		branch = gitMgr.GetDefaultBranch(userRepo)
	}

	fmt.Println(tui.RenderInfo("Checking for SKILL.md..."))
	isSingleSkill := gitMgr.CheckRemoteSkillMd(userRepo, branch, gitInfo.Path)

	if isSingleSkill {
		return addSingleGitHubSkill(cm, registry, gitMgr, gitInfo, userRepo, branch)
	}

	// No SKILL.md at root — might be a folder of skills. Clone and scan.
	return addGitHubSkillsFolder(cm, registry, gitMgr, gitInfo, userRepo, branch)
}

// addSingleGitHubSkill handles a GitHub URL pointing to a single skill (has SKILL.md).
func addSingleGitHubSkill(cm *config.Manager, registry *skills.Registry, gitMgr *git.Manager, gitInfo git.URLInfo, userRepo, branch string) []string {
	id := "github:" + userRepo
	if gitInfo.Path != "" {
		id += "/" + gitInfo.Path
	}

	if existing := registry.GetSkill(id); existing != nil {
		var overwrite bool
		if err := huh.NewForm(huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("%s already exists. Overwrite?", id)).
				Value(&overwrite),
		)).Run(); err != nil {
			return nil
		}
		if !overwrite {
			return nil
		}
		os.RemoveAll(cm.GetRepoPath(id))
	}

	destPath := cm.GetRepoPath(id)
	os.MkdirAll(filepath.Dir(destPath), 0755)

	fmt.Println(tui.RenderInfo("Cloning " + id + "..."))

	var err error
	if gitInfo.Path != "" {
		err = gitMgr.CloneSparseQuiet(gitInfo.URL, destPath, gitInfo.Path, branch)
	} else {
		err = gitMgr.CloneFullQuiet(gitInfo.URL, destPath)
	}
	if err != nil {
		fmt.Println(tui.RenderError("Failed to clone: " + err.Error()))
		return nil
	}

	subPath := "."
	if gitInfo.Path != "" {
		subPath = gitInfo.Path
	}
	commitID, _ := gitMgr.GetLocalPathCommitID(destPath, subPath)

	registry.AddSkill(id, "github", commitID, gitInfo.Path)
	fmt.Println(tui.RenderSuccess("Added " + id))
	return []string{id}
}

// addGitHubSkillsFolder handles a GitHub URL pointing to a folder of skills (no SKILL.md at root).
// Clones the path, scans for subdirectories with SKILL.md, and lets the user pick.
func addGitHubSkillsFolder(cm *config.Manager, registry *skills.Registry, gitMgr *git.Manager, gitInfo git.URLInfo, userRepo, branch string) []string {
	fmt.Println(tui.RenderInfo("No SKILL.md at root — scanning for skills inside..."))

	// Clone to a temp location to scan
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("agm-scan-%d", os.Getpid()))
	defer os.RemoveAll(tmpDir)

	var err error
	if gitInfo.Path != "" {
		err = gitMgr.CloneSparseQuiet(gitInfo.URL, tmpDir, gitInfo.Path, branch)
	} else {
		err = gitMgr.CloneFullQuiet(gitInfo.URL, tmpDir)
	}
	if err != nil {
		fmt.Println(tui.RenderError("Failed to clone: " + err.Error()))
		return nil
	}

	// Scan for skills
	scanRoot := tmpDir
	if gitInfo.Path != "" {
		scanRoot = filepath.Join(tmpDir, gitInfo.Path)
	}

	type skillEntry struct {
		name string
	}
	var found []skillEntry

	entries, err := os.ReadDir(scanRoot)
	if err != nil {
		fmt.Println(tui.RenderError("Error reading directory: " + err.Error()))
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if _, err := os.Stat(filepath.Join(scanRoot, entry.Name(), "SKILL.md")); err == nil {
			found = append(found, skillEntry{name: entry.Name()})
		}
	}

	if len(found) == 0 {
		fmt.Println(tui.RenderError("No skills found (no subdirectories with SKILL.md)."))
		return nil
	}

	// Let user pick
	var opts []huh.Option[string]
	for _, s := range found {
		id := "github:" + userRepo
		if gitInfo.Path != "" {
			id += "/" + gitInfo.Path
		}
		id += "/" + s.name
		label := s.name
		if existing := registry.GetSkill(id); existing != nil {
			label += " " + tui.MutedText.Render("(installed)")
		}
		opts = append(opts, huh.NewOption(label, s.name))
	}

	var selected []string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title(fmt.Sprintf("Found %d skills — select which to add", len(found))).
			Options(opts...).
			Value(&selected),
	)).Run(); err != nil {
		return nil
	}

	if len(selected) == 0 {
		fmt.Println(tui.MutedText.Render("No skills selected."))
		return nil
	}

	var addedIDs []string
	for _, selName := range selected {
		var match *skillEntry
		for i := range found {
			if found[i].name == selName {
				match = &found[i]
				break
			}
		}
		if match == nil {
			continue
		}

		id := "github:" + userRepo
		if gitInfo.Path != "" {
			id += "/" + gitInfo.Path
		}
		id += "/" + match.name

		skillSubPath := match.name
		if gitInfo.Path != "" {
			skillSubPath = gitInfo.Path + "/" + match.name
		}

		if existing := registry.GetSkill(id); existing != nil {
			var overwrite bool
			if err := huh.NewForm(huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("%s already exists. Overwrite?", id)).
					Value(&overwrite),
			)).Run(); err != nil {
				continue
			}
			if !overwrite {
				continue
			}
			os.RemoveAll(cm.GetRepoPath(id))
		}

		destPath := cm.GetRepoPath(id)
		os.MkdirAll(filepath.Dir(destPath), 0755)
		fmt.Println(tui.RenderInfo("Cloning " + match.name + "..."))
		if err := gitMgr.CloneSparseQuiet(gitInfo.URL, destPath, skillSubPath, branch); err != nil {
			fmt.Println(tui.RenderError("Failed to clone " + match.name + ": " + err.Error()))
			continue
		}

		commitID, _ := gitMgr.GetLocalPathCommitID(destPath, skillSubPath)
		registry.AddSkill(id, "github", commitID, skillSubPath)
		addedIDs = append(addedIDs, id)
		fmt.Println(tui.RenderSuccess("Added " + id))
	}

	if len(addedIDs) > 0 {
		fmt.Printf("\n%s\n", tui.RenderSuccess(fmt.Sprintf("%d skill(s) added", len(addedIDs))))
	}
	return addedIDs
}

// addSkillsFolder scans a directory and lets the user pick skills. Returns added IDs.
func addSkillsFolder() []string {
	cm, err := config.NewManager()
	if err != nil {
		fmt.Println(tui.RenderError("Failed to initialize config: " + err.Error()))
		return nil
	}
	registry := skills.NewRegistry(cm)

	var inputPath string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Skills folder path").
				Description("Directory containing skill subdirectories").
				Value(&inputPath),
		),
	)
	if err := form.Run(); err != nil || inputPath == "" {
		return nil
	}

	folderPath := resolvePath(strings.TrimSpace(inputPath))
	if folderPath == "" {
		return nil
	}

	dirInfo, err := os.Stat(folderPath)
	if err != nil || !dirInfo.IsDir() {
		fmt.Println(tui.RenderError("Path does not exist or is not a directory."))
		return nil
	}

	entries, err := os.ReadDir(folderPath)
	if err != nil {
		fmt.Println(tui.RenderError("Error reading directory: " + err.Error()))
		return nil
	}

	type skillEntry struct {
		name string
		path string
	}
	var found []skillEntry

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(folderPath, entry.Name(), "SKILL.md")); err == nil {
			found = append(found, skillEntry{name: entry.Name(), path: filepath.Join(folderPath, entry.Name())})
		}
	}

	if len(found) == 0 {
		fmt.Println(tui.RenderWarning("No skills found (no subdirectories with SKILL.md)."))
		return nil
	}

	var opts []huh.Option[string]
	for _, s := range found {
		id := "local:" + s.name
		label := s.name
		if existing := registry.GetSkill(id); existing != nil {
			label += " " + tui.MutedText.Render("(installed)")
		}
		opts = append(opts, huh.NewOption(label, s.name))
	}

	var selected []string
	if err := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title(fmt.Sprintf("Found %d skills — select which to add", len(found))).
			Options(opts...).
			Value(&selected),
	)).Run(); err != nil {
		return nil
	}

	if len(selected) == 0 {
		fmt.Println(tui.MutedText.Render("No skills selected."))
		return nil
	}

	var addedIDs []string
	for _, selName := range selected {
		var match *skillEntry
		for i := range found {
			if found[i].name == selName {
				match = &found[i]
				break
			}
		}
		if match == nil {
			continue
		}

		id := "local:" + match.name

		if existing := registry.GetSkill(id); existing != nil {
			var overwrite bool
			if err := huh.NewForm(huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("%s already exists. Overwrite?", id)).
					Value(&overwrite),
			)).Run(); err != nil {
				continue
			}
			if !overwrite {
				continue
			}
			os.RemoveAll(cm.GetRepoPath(id))
		}

		destPath := cm.GetRepoPath(id)
		fmt.Println(tui.RenderInfo("Copying " + match.name + "..."))
		if err := copyDir(match.path, destPath); err != nil {
			fmt.Println(tui.RenderError("Failed: " + match.name + ": " + err.Error()))
			continue
		}

		registry.AddSkill(id, "local", "", "")
		addedIDs = append(addedIDs, id)
		fmt.Println(tui.RenderSuccess("Added " + id))
	}

	if len(addedIDs) > 0 {
		fmt.Printf("\n%s\n", tui.RenderSuccess(fmt.Sprintf("%d skill(s) added", len(addedIDs))))
	}
	return addedIDs
}

// --- helpers ---

func resolvePath(inputPath string) string {
	if inputPath == "" {
		return ""
	}
	if strings.HasPrefix(inputPath, "~/") {
		home, _ := os.UserHomeDir()
		inputPath = filepath.Join(home, inputPath[2:])
	} else if inputPath == "~" {
		inputPath, _ = os.UserHomeDir()
	}
	abs, err := filepath.Abs(inputPath)
	if err != nil {
		fmt.Println(tui.RenderError("Cannot resolve path: " + err.Error()))
		return ""
	}
	return abs
}

func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if err := copyFile(srcPath, dstPath, info.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}

	return out.Close()
}
