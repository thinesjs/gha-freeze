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

type State int

const (
	StateLoading State = iota
	StateFileSelection
	StateScanning
	StateActionReview
	StateResolving
	StateConfirming
	StateProcessing
	StateComplete
	StateBackupList
	StateRestoring
	StateError
	StateRateLimited
)

type Model struct {
	state         State
	spinner       spinner.Model
	workflowFiles []string
	selectedFiles []string
	fileList      list.Model
	backupList    list.Model
	actions       []workflow.ActionReference
	replacements  []workflow.Replacement
	err           error
	githubClient  *github.Client
	backupPath    string
	totalCount    int
	dryRun        bool
	noBackup      bool
	message       string
	tokenPrompt   bool
	tokenInput    string
	version       string
}

type workflowFileItem struct {
	title   string
	checked bool
}

func (i workflowFileItem) Title() string {
	checkbox := "[ ]"
	if i.checked {
		checkbox = "[✓]"
	}
	return checkbox + " " + i.title
}
func (i workflowFileItem) Description() string { return "" }
func (i workflowFileItem) FilterValue() string { return i.title }

type loadingCompleteMsg struct {
	files []string
	err   error
}

type scanCompleteMsg struct {
	actions []workflow.ActionReference
	err     error
}

type resolveCompleteMsg struct {
	replacements []workflow.Replacement
	err          error
}

type processCompleteMsg struct {
	backupPath string
	err        error
}

type rateLimitMsg struct{}

type backupListMsg struct {
	backups []backup.BackupInfo
	err     error
}

type restoreCompleteMsg struct {
	err error
}

type backupItem struct {
	info backup.BackupInfo
}

func (i backupItem) Title() string {
	return fmt.Sprintf("%s (%d files)", i.info.Timestamp, i.info.FileCount)
}
func (i backupItem) Description() string { return i.info.Path }
func (i backupItem) FilterValue() string { return i.info.Timestamp }

func NewModel(token string, dryRun, noBackup bool, version string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return Model{
		state:        StateLoading,
		spinner:      s,
		githubClient: github.NewClient(token),
		dryRun:       dryRun,
		noBackup:     noBackup,
		version:      version,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		loadWorkflowFiles,
	)
}

func loadWorkflowFiles() tea.Msg {
	files, err := workflow.FindWorkflowFiles()
	return loadingCompleteMsg{files: files, err: err}
}
