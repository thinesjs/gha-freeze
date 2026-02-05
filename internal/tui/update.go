package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thinesjs/gha-freeze/internal/backup"
	"github.com/thinesjs/gha-freeze/internal/github"
	"github.com/thinesjs/gha-freeze/internal/workflow"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == StateFileSelection {
			switch msg.String() {
			case "enter":
				return m.handleEnterKey()
			case "ctrl+c", "q":
				return m, tea.Quit
			case " ":
				return m.toggleFileSelection()
			default:
				var cmd tea.Cmd
				m.fileList, cmd = m.fileList.Update(msg)
				return m, cmd
			}
		}
		return m.handleKeyPress(msg)

	case loadingCompleteMsg:
		return m.handleLoadingComplete(msg)

	case scanCompleteMsg:
		return m.handleScanComplete(msg)

	case resolveCompleteMsg:
		return m.handleResolveComplete(msg)

	case processCompleteMsg:
		return m.handleProcessComplete(msg)

	case rateLimitMsg:
		return m.handleRateLimit()

	case backupListMsg:
		return m.handleBackupList(msg)

	case restoreCompleteMsg:
		return m.handleRestoreComplete(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	if m.state == StateFileSelection {
		var cmd tea.Cmd
		m.fileList, cmd = m.fileList.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state == StateComplete {
		switch msg.String() {
		case "d":
			if m.backupPath != "" {
				return m.handleDeleteBackup()
			}
		case "k", "q":
			return m, tea.Quit
		case "r":
			return m.loadBackupList()
		}
		return m, nil
	}

	if m.state == StateBackupList {
		switch msg.String() {
		case "enter":
			return m.handleRestoreBackup()
		case "q", "esc":
			m.state = StateComplete
			return m, nil
		default:
			var cmd tea.Cmd
			m.backupList, cmd = m.backupList.Update(msg)
			return m, cmd
		}
	}

	if m.state == StateError {
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		if m.state != StateProcessing {
			return m, tea.Quit
		}

	case "enter":
		return m.handleEnterKey()

	case " ":
		if m.state == StateFileSelection {
			return m.toggleFileSelection()
		}
	}

	if m.state == StateRateLimited && m.tokenPrompt {
		if msg.String() == "enter" {
			m.githubClient.SetToken(m.tokenInput)
			m.state = StateActionReview
			m.tokenPrompt = false
			return m, m.resolveActions()
		} else if msg.String() == "backspace" {
			if len(m.tokenInput) > 0 {
				m.tokenInput = m.tokenInput[:len(m.tokenInput)-1]
			}
		} else if len(msg.String()) == 1 {
			m.tokenInput += msg.String()
		}
	}

	return m, nil
}

func (m Model) handleEnterKey() (tea.Model, tea.Cmd) {
	switch m.state {
	case StateFileSelection:
		m.selectedFiles = m.getSelectedFiles()
		if len(m.selectedFiles) == 0 {
			m.err = fmt.Errorf("no files selected")
			m.state = StateError
			return m, nil
		}
		m.state = StateScanning
		return m, tea.Batch(m.spinner.Tick, m.scanFiles())

	case StateActionReview:
		if len(m.actions) == 0 {
			m.message = "No actions to pin"
			m.state = StateComplete
			return m, nil
		}
		m.state = StateResolving
		return m, tea.Batch(m.spinner.Tick, m.resolveActions())

	case StateConfirming:
		m.state = StateProcessing
		return m, tea.Batch(m.spinner.Tick, m.processActions())

	case StateRateLimited:
		m.tokenPrompt = true
		return m, nil
	}

	return m, nil
}

func (m Model) handleLoadingComplete(msg loadingCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.state = StateError
		return m, nil
	}

	m.workflowFiles = msg.files
	if len(m.workflowFiles) == 0 {
		m.err = fmt.Errorf("no workflow files found in .github/workflows")
		m.state = StateError
		return m, nil
	}

	items := make([]list.Item, len(m.workflowFiles))
	for i, f := range m.workflowFiles {
		items[i] = workflowFileItem{title: f, checked: true}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetHeight(1)
	delegate.SetSpacing(0)

	listHeight := len(m.workflowFiles) + 2
	if listHeight > 15 {
		listHeight = 15
	}

	m.fileList = list.New(items, delegate, 80, listHeight)
	m.fileList.SetShowTitle(false)
	m.fileList.SetShowStatusBar(false)
	m.fileList.SetFilteringEnabled(false)
	m.fileList.SetShowHelp(false)
	m.state = StateFileSelection
	return m, nil
}

func (m Model) handleScanComplete(msg scanCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.state = StateError
		return m, nil
	}

	var unpinnedActions []workflow.ActionReference
	for _, action := range msg.actions {
		if !action.IsPinned {
			unpinnedActions = append(unpinnedActions, action)
		}
	}

	m.actions = unpinnedActions
	m.state = StateActionReview
	return m, nil
}

func (m Model) handleResolveComplete(msg resolveCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		if github.IsRateLimitError(msg.err) {
			m.state = StateRateLimited
			m.err = msg.err
			return m, nil
		}
		m.err = msg.err
		m.state = StateError
		return m, nil
	}

	m.replacements = msg.replacements
	m.totalCount = len(m.replacements)
	m.state = StateConfirming
	return m, nil
}

func (m Model) handleProcessComplete(msg processCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.state = StateError
		return m, nil
	}

	m.backupPath = msg.backupPath
	m.state = StateComplete
	return m, nil
}

func (m Model) handleRateLimit() (tea.Model, tea.Cmd) {
	m.state = StateRateLimited
	return m, nil
}

func (m Model) handleDeleteBackup() (tea.Model, tea.Cmd) {
	if m.backupPath != "" {
		if err := backup.DeleteBackup(m.backupPath); err != nil {
			m.message = fmt.Sprintf("Failed to delete backup: %s", err.Error())
		} else {
			m.message = fmt.Sprintf("Deleted backup: %s", m.backupPath)
			m.backupPath = ""
		}
	}
	return m, nil
}

func (m Model) loadBackupList() (tea.Model, tea.Cmd) {
	return m, func() tea.Msg {
		backups, err := backup.ListBackups()
		return backupListMsg{backups: backups, err: err}
	}
}

func (m Model) handleBackupList(msg backupListMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.state = StateError
		return m, nil
	}

	if len(msg.backups) == 0 {
		m.message = "No backups found"
		m.state = StateComplete
		return m, nil
	}

	items := make([]list.Item, len(msg.backups))
	for i, b := range msg.backups {
		items[i] = backupItem{info: b}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.SetHeight(2)
	delegate.SetSpacing(0)

	listHeight := (len(msg.backups) * 2) + 2
	if listHeight > 15 {
		listHeight = 15
	}

	m.backupList = list.New(items, delegate, 80, listHeight)
	m.backupList.SetShowTitle(false)
	m.backupList.SetShowStatusBar(false)
	m.backupList.SetFilteringEnabled(false)
	m.backupList.SetShowHelp(false)
	m.state = StateBackupList
	return m, nil
}

func (m Model) handleRestoreBackup() (tea.Model, tea.Cmd) {
	selectedIdx := m.backupList.Index()
	if selectedIdx >= 0 && selectedIdx < len(m.backupList.Items()) {
		item := m.backupList.Items()[selectedIdx]
		if bItem, ok := item.(backupItem); ok {
			m.state = StateRestoring
			return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
				err := backup.RestoreBackup(bItem.info.Path)
				return restoreCompleteMsg{err: err}
			})
		}
	}
	return m, nil
}

func (m Model) handleRestoreComplete(msg restoreCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.state = StateError
		return m, nil
	}

	m.message = "Backup restored successfully!"
	m.state = StateComplete
	return m, nil
}

func (m Model) toggleFileSelection() (tea.Model, tea.Cmd) {
	selectedIdx := m.fileList.Index()
	if selectedIdx >= 0 && selectedIdx < len(m.fileList.Items()) {
		item := m.fileList.Items()[selectedIdx]
		if wfItem, ok := item.(workflowFileItem); ok {
			wfItem.checked = !wfItem.checked
			m.fileList.SetItem(selectedIdx, wfItem)
		}
	}
	return m, nil
}

func (m Model) getSelectedFiles() []string {
	var selected []string
	for _, item := range m.fileList.Items() {
		if wfItem, ok := item.(workflowFileItem); ok && wfItem.checked {
			selected = append(selected, wfItem.title)
		}
	}
	if len(selected) == 0 {
		selected = m.workflowFiles
	}
	return selected
}

func (m Model) scanFiles() tea.Cmd {
	return func() tea.Msg {
		var allActions []workflow.ActionReference
		for _, file := range m.selectedFiles {
			actions, err := workflow.ParseWorkflowFile(file)
			if err != nil {
				return scanCompleteMsg{err: err}
			}
			allActions = append(allActions, actions...)
		}
		return scanCompleteMsg{actions: allActions}
	}
}

func (m Model) resolveActions() tea.Cmd {
	return func() tea.Msg {
		var replacements []workflow.Replacement
		var lastError error

		for _, action := range m.actions {
			resolved := m.githubClient.ResolveAction(action.Owner, action.Repo, action.Ref)
			if resolved.Error != nil {
				lastError = resolved.Error
				if github.IsRateLimitError(resolved.Error) {
					return rateLimitMsg{}
				}
				continue
			}

			replacements = append(replacements, workflow.Replacement{
				Action:  action,
				SHA:     resolved.SHA,
				Version: resolved.Version,
			})
		}

		if len(replacements) == 0 && lastError != nil {
			return resolveCompleteMsg{err: lastError}
		}

		return resolveCompleteMsg{replacements: replacements}
	}
}

func (m Model) processActions() tea.Cmd {
	return func() tea.Msg {
		var backupPath string
		var err error

		if !m.noBackup && !m.dryRun {
			backupPath, err = backup.CreateBackup(m.selectedFiles)
			if err != nil {
				return processCompleteMsg{err: err}
			}
		}

		if !m.dryRun {
			fileReplacements := make(map[string][]workflow.Replacement)
			for _, repl := range m.replacements {
				fileReplacements[repl.Action.FilePath] = append(fileReplacements[repl.Action.FilePath], repl)
			}

			for file, repls := range fileReplacements {
				if err := workflow.ReplaceActionsInFile(file, repls); err != nil {
					return processCompleteMsg{err: err}
				}
			}
		}

		return processCompleteMsg{backupPath: backupPath}
	}
}
