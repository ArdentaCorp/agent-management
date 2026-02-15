package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary   = lipgloss.Color("#7C3AED") // purple
	Secondary = lipgloss.Color("#06B6D4") // cyan
	Success   = lipgloss.Color("#22C55E") // green
	Warning   = lipgloss.Color("#F59E0B") // amber
	Error     = lipgloss.Color("#EF4444") // red
	Muted     = lipgloss.Color("#6B7280") // gray
	White     = lipgloss.Color("#FFFFFF")

	// Text styles
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary)

	Subtitle = lipgloss.NewStyle().
			Foreground(Secondary)

	SuccessText = lipgloss.NewStyle().
			Foreground(Success)

	ErrorText = lipgloss.NewStyle().
			Foreground(Error)

	WarningText = lipgloss.NewStyle().
			Foreground(Warning)

	MutedText = lipgloss.NewStyle().
			Foreground(Muted)

	// Banner
	Banner = lipgloss.NewStyle().
		Bold(true).
		Foreground(White).
		Background(Primary).
		Padding(0, 1)

	// Section header
	SectionHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(Primary).
			MarginBottom(1)

	// Skill item
	SkillItem = lipgloss.NewStyle().
			PaddingLeft(2)

	SkillLinked = lipgloss.NewStyle().
			Foreground(Success).
			PaddingLeft(2)

	SkillUnlinked = lipgloss.NewStyle().
			Foreground(Muted).
			PaddingLeft(2)
)

// RenderBanner returns the styled app banner.
func RenderBanner(version string) string {
	banner := Banner.Render(" agm ") + " " + Title.Render("Agent Management") + " " + MutedText.Render("v"+version)
	return "\n" + banner + "\n"
}

// RenderSection returns a styled section header.
func RenderSection(title string) string {
	return "\n" + SectionHeader.Render(title) + "\n"
}

// RenderSuccess returns a styled success message.
func RenderSuccess(msg string) string {
	return SuccessText.Render("✓ ") + msg
}

// RenderError returns a styled error message.
func RenderError(msg string) string {
	return ErrorText.Render("✗ ") + msg
}

// RenderWarning returns a styled warning message.
func RenderWarning(msg string) string {
	return WarningText.Render("! ") + msg
}

// RenderInfo returns a styled info message.
func RenderInfo(msg string) string {
	return Subtitle.Render("→ ") + msg
}
