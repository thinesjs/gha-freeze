package main

import (
	"fmt"
	"net/url"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/thinesjs/gha-freeze/internal/config"
	"github.com/thinesjs/gha-freeze/internal/github"
	"github.com/thinesjs/gha-freeze/internal/tui"
	"github.com/thinesjs/gha-freeze/internal/updater"
)

var (
	version       = "dev"
	token         string
	dryRun        bool
	noBackup      bool
	checkUpdate   bool
	skipUpdateChk bool
)

var rootCmd = &cobra.Command{
	Use:   "gha-freeze",
	Short: "Pin GitHub Actions to specific SHA commits",
	Long: `gha-freeze is a CLI tool that pins GitHub Actions in your workflows to specific
SHA commits with version comments for security and reproducibility.

It will:
  1. Find all workflow files in .github/workflows
  2. Parse them for action references
  3. Resolve versions to SHA commits via GitHub API
  4. Create backups before modifying files
  5. Replace action references with SHA + version comments`,
	RunE: run,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gha-freeze version %s\n", version)
	},
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update to the latest version",
	RunE:  runUpdate,
}

var authCmd = &cobra.Command{
	Use:   "auth [token]",
	Short: "Save GitHub token for future use",
	Long: `Save a GitHub Personal Access Token for automatic use in future commands.
The token is stored securely in ~/.config/gha-freeze/token with 0600 permissions.

Alternatively, you can set the GITHUB_TOKEN or GHA_FREEZE_TOKEN environment variable.`,
	Args: cobra.ExactArgs(1),
	RunE: runAuth,
}

func init() {
	rootCmd.Flags().StringVar(&token, "token", "", "GitHub token (create at: https://github.com/settings/tokens/new?description=gha-freeze&scopes=public_repo)")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without modifying files")
	rootCmd.Flags().BoolVar(&noBackup, "no-backup", false, "Skip creating backup files")
	rootCmd.Flags().BoolVar(&checkUpdate, "check-update", false, "Check for updates without installing")
	rootCmd.Flags().BoolVar(&skipUpdateChk, "skip-update-check", false, "Skip automatic update check on startup")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuth(cmd *cobra.Command, args []string) error {
	token := args[0]
	if err := config.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	tokenPath, _ := config.GetTokenPath()
	fmt.Printf("✓ Token saved to %s\n", tokenPath)
	fmt.Printf("The token will be used automatically for future commands.\n")
	return nil
}

func run(cmd *cobra.Command, args []string) error {
	if checkUpdate {
		return checkForUpdates()
	}

	if !skipUpdateChk {
		_ = checkAndNotifyUpdate()
	}

	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository. Please run this command from the root of a git repository")
	}

	if _, err := os.Stat(".github/workflows"); os.IsNotExist(err) {
		return fmt.Errorf(".github/workflows directory not found")
	}

	resolvedToken := config.GetToken(token)

	m := tui.NewModel(resolvedToken, dryRun, noBackup, version)
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Printf("Checking for updates...\n")

	resolvedToken := config.GetToken(token)
	info, err := updater.CheckForUpdateWithToken(version, resolvedToken)
	if err != nil {
		if github.IsRateLimitError(err) {
			fmt.Printf("\nGitHub API rate limit reached.\n\n")
			fmt.Printf("Create a token to get higher rate limits:\n")
			fmt.Printf("%s\n\n", getTokenCreationURL())
			fmt.Printf("Then save it: gha-freeze auth YOUR_TOKEN\n")
			return nil
		}
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !info.Available {
		fmt.Printf("You are already on the latest version (%s)\n", info.CurrentVersion)
		return nil
	}

	fmt.Printf("New version available: %s -> %s\n", info.CurrentVersion, info.LatestVersion)
	if info.ReleaseNotes != "" {
		fmt.Printf("\nRelease Notes:\n%s\n\n", info.ReleaseNotes)
	}

	if info.DownloadURL == "" {
		fmt.Printf("Automatic update not available for your platform.\n")
		fmt.Printf("Download manually from: https://github.com/thinesjs/gha-freeze/releases/latest\n")
		return nil
	}

	fmt.Printf("Downloading and installing update...\n")
	if err := updater.DownloadAndInstall(info.DownloadURL); err != nil {
		return fmt.Errorf("failed to install update: %w", err)
	}

	fmt.Printf("✓ Successfully updated to version %s\n", info.LatestVersion)
	fmt.Printf("Please restart gha-freeze to use the new version\n")

	return nil
}

func checkForUpdates() error {
	resolvedToken := config.GetToken(token)
	info, err := updater.CheckForUpdateWithToken(version, resolvedToken)
	if err != nil {
		if github.IsRateLimitError(err) {
			fmt.Printf("GitHub API rate limit reached.\n\n")
			fmt.Printf("Create a token to get higher rate limits:\n")
			fmt.Printf("%s\n\n", getTokenCreationURL())
			fmt.Printf("Then save it: gha-freeze auth YOUR_TOKEN\n")
			return nil
		}
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if info.Available {
		fmt.Printf("New version available: %s -> %s\n", info.CurrentVersion, info.LatestVersion)
		if info.DownloadURL != "" {
			fmt.Printf("Run 'gha-freeze update' to install\n")
		} else {
			fmt.Printf("Download from: https://github.com/thinesjs/gha-freeze/releases/latest\n")
		}
	} else {
		fmt.Printf("You are on the latest version (%s)\n", info.CurrentVersion)
	}

	return nil
}

func getTokenCreationURL() string {
	timestamp := time.Now().Format("2006-01-02")
	description := fmt.Sprintf("gha-freeze-%s", timestamp)
	return fmt.Sprintf("https://github.com/settings/tokens/new?description=%s&scopes=public_repo",
		url.QueryEscape(description))
}

func checkAndNotifyUpdate() error {
	resolvedToken := config.GetToken(token)
	info, err := updater.CheckForUpdateWithToken(version, resolvedToken)
	if err != nil {
		return err
	}

	if info.Available {
		fmt.Printf("\n")
		fmt.Printf("╭─────────────────────────────────────────────────╮\n")
		fmt.Printf("│  New version available: %s -> %s", info.CurrentVersion, info.LatestVersion)
		padding := 49 - 24 - len(info.CurrentVersion) - len(info.LatestVersion)
		for i := 0; i < padding; i++ {
			fmt.Printf(" ")
		}
		fmt.Printf("│\n")
		fmt.Printf("│  Run 'gha-freeze update' to install             │\n")
		fmt.Printf("╰─────────────────────────────────────────────────╯\n")
		fmt.Printf("\n")
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
