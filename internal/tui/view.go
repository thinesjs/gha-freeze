package tui

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	warningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))
)

func (m Model) View() string {
	switch m.state {
	case StateLoading:
		return m.viewLoading()
	case StateFileSelection:
		return m.viewFileSelection()
	case StateScanning:
		return m.viewScanning()
	case StateActionReview:
		return m.viewActionReview()
	case StateResolving:
		return m.viewResolving()
	case StateConfirming:
		return m.viewConfirming()
	case StateProcessing:
		return m.viewProcessing()
	case StateComplete:
		return m.viewComplete()
	case StateBackupList:
		return m.viewBackupList()
	case StateRestoring:
		return m.viewRestoring()
	case StateError:
		return m.viewError()
	case StateRateLimited:
		return m.viewRateLimited()
	default:
		return "Unknown state"
	}
}

func (m Model) viewLoading() string {
	return fmt.Sprintf("\n%s %s\n", m.spinner.View(), "Finding workflow files...")
}

func (m Model) viewFileSelection() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Select Workflow Files") + "\n\n")
	b.WriteString(m.fileList.View())
	b.WriteString("\n\n" + infoStyle.Render("↑/↓: navigate • space: toggle selection • enter: continue • q: quit"))
	return b.String()
}

func (m Model) viewScanning() string {
	return fmt.Sprintf("\n%s %s\n", m.spinner.View(), "Scanning workflow files for actions...")
}

func (m Model) viewResolving() string {
	return fmt.Sprintf("\n%s Resolving %d actions to SHA commits via GitHub API...\n", m.spinner.View(), len(m.actions))
}

func (m Model) viewActionReview() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Actions Found") + "\n\n")

	if len(m.actions) == 0 {
		b.WriteString(infoStyle.Render("No unpinned actions found. All actions are already pinned!") + "\n")
		b.WriteString("\n" + infoStyle.Render("Press Enter to exit"))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Found %d unpinned actions:\n\n", len(m.actions)))

	for _, action := range m.actions {
		b.WriteString(fmt.Sprintf("  %s/%s@%s (%s:%d)\n",
			action.Owner, action.Repo, action.Ref, action.FilePath, action.Line))
	}

	b.WriteString("\n" + infoStyle.Render("Press Enter to resolve and pin these actions, q to quit"))
	return b.String()
}

func (m Model) viewConfirming() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Ready to Pin Actions") + "\n\n")

	if len(m.replacements) == 0 {
		b.WriteString(warningStyle.Render("No actions could be resolved") + "\n")
		b.WriteString("\n" + infoStyle.Render("Press Enter to exit"))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Will pin %d actions:\n\n", len(m.replacements)))

	for _, repl := range m.replacements {
		b.WriteString(fmt.Sprintf("  %s/%s@%s\n    → %s/%s@%s # %s\n\n",
			repl.Action.Owner, repl.Action.Repo, repl.Action.Ref,
			repl.Action.Owner, repl.Action.Repo, repl.SHA, repl.Version))
	}

	if m.dryRun {
		b.WriteString(warningStyle.Render("DRY RUN MODE - No changes will be made") + "\n")
	}

	b.WriteString("\n" + infoStyle.Render("Press Enter to confirm, q to cancel"))
	return b.String()
}

func (m Model) viewProcessing() string {
	return fmt.Sprintf("\n%s Processing actions...\n", m.spinner.View())
}

func (m Model) viewComplete() string {
	var b strings.Builder
	b.WriteString(successStyle.Render("✓ Complete!") + "\n\n")

	if m.dryRun {
		b.WriteString(fmt.Sprintf("Would have pinned %d actions\n", len(m.replacements)))
	} else {
		b.WriteString(fmt.Sprintf("Pinned %d actions\n", len(m.replacements)))

		if m.backupPath != "" {
			b.WriteString(fmt.Sprintf("Backup created at: %s\n\n", m.backupPath))
		}
	}

	if m.message != "" {
		b.WriteString("\n" + successStyle.Render(m.message) + "\n")
	}

	b.WriteString("\n" + infoStyle.Render("Options:") + "\n")
	if m.backupPath != "" && !m.dryRun {
		b.WriteString("  d - Delete this backup\n")
		b.WriteString("  k - Keep this backup\n")
	}
	b.WriteString("  r - Restore from backup\n")
	b.WriteString("  q - Quit\n")

	return b.String()
}

func (m Model) viewBackupList() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Restore from Backup") + "\n\n")
	b.WriteString(m.backupList.View())
	b.WriteString("\n\n" + infoStyle.Render("↑/↓: navigate • enter: restore selected • q: cancel"))
	return b.String()
}

func (m Model) viewRestoring() string {
	return fmt.Sprintf("\n%s Restoring workflow files from backup...\n", m.spinner.View())
}

func (m Model) viewError() string {
	var b strings.Builder
	b.WriteString(errorStyle.Render("Error") + "\n\n")
	b.WriteString(fmt.Sprintf("%s\n", m.err.Error()))
	b.WriteString("\n" + infoStyle.Render("Press q to quit"))
	return b.String()
}

func (m Model) viewRateLimited() string {
	var b strings.Builder
	b.WriteString(warningStyle.Render("GitHub API Rate Limit Reached") + "\n\n")

	if m.tokenPrompt {
		b.WriteString("Enter GitHub Personal Access Token:\n")
		b.WriteString(strings.Repeat("*", len(m.tokenInput)) + "\n")
		b.WriteString("\n" + infoStyle.Render("Press Enter when done, Ctrl+C to cancel"))
	} else {
		timestamp := time.Now().Format("2006-01-02")
		description := fmt.Sprintf("gha-freeze-%s", timestamp)
		tokenURL := fmt.Sprintf("https://github.com/settings/tokens/new?description=%s&scopes=public_repo",
			url.QueryEscape(description))

		b.WriteString("You've hit the GitHub API rate limit for unauthenticated requests.\n\n")
		b.WriteString("Create a token with public_repo scope:\n")
		b.WriteString(tokenURL + "\n\n")
		b.WriteString("Save it for future use: gha-freeze auth YOUR_TOKEN\n\n")
		b.WriteString(infoStyle.Render("Press Enter to provide a GitHub token, q to quit"))
	}

	return b.String()
}
