package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ArdentaCorp/agent-management/internal/commands"
	"github.com/ArdentaCorp/agent-management/internal/config"
	"github.com/ArdentaCorp/agent-management/internal/tui"
	"github.com/charmbracelet/huh"
)

const version = "1.0.0"

func main() {
	args := os.Args[1:]

	if len(args) > 0 {
		for _, arg := range args {
			switch arg {
			case "--version", "-v":
				fmt.Printf("agm version %s\n", version)
				return
			case "--config":
				showConfig()
				return
			case "--sync":
				fmt.Print(tui.RenderBanner(version))
				commands.SyncSkills(false)
				return
			case "--help", "-h":
				printHelp()
				return
			default:
				fmt.Fprintln(os.Stderr, tui.RenderError("Unknown argument: "+arg))
				printHelp()
				os.Exit(1)
			}
		}
		return
	}

	mainMenu()
}

func printHelp() {
	fmt.Println(tui.RenderBanner(version))
	fmt.Println("Usage: agm [options]")
	fmt.Println()
	fmt.Println("  A CLI tool to manage and synchronize AI coding agent skills")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --version, -v  Show version number")
	fmt.Println("  --config       Show configuration")
	fmt.Println("  --sync         Sync skills from registry (non-interactive)")
	fmt.Println("  --help, -h     Show this help message")
	fmt.Println()
	fmt.Println("Run without arguments for interactive mode.")
}

func showConfig() {
	cm := config.NewManager()
	fmt.Println(tui.RenderBanner(version))
	fmt.Println(tui.RenderInfo("Config directory: " + cm.GetHomeDir()))

	cfg, err := cm.LoadConfig()
	if err != nil {
		return
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Printf("\n%s\n", string(data))
}

func mainMenu() {
	for {
		fmt.Print(tui.RenderBanner(version))

		var selected string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("What would you like to do?").
					Options(
						huh.NewOption("📥 Import skills", "add"),
						huh.NewOption("🔗 Link to project", "link"),
						huh.NewOption("⚙️  Manage skills", "manage"),
						huh.NewOption("👋 Exit", "exit"),
					).
					Value(&selected),
			),
		)

		if err := form.Run(); err != nil {
			return
		}

		switch selected {
		case "exit":
			fmt.Println(tui.MutedText.Render("\nGoodbye! 👋"))
			return
		case "add":
			commands.AddSkills()
		case "link":
			commands.LinkToProject()
		case "manage":
			commands.ManageSkills()
		}
	}
}
