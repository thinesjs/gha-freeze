package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func CreateBackup(files []string) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupDir := filepath.Join(".github", "workflows", fmt.Sprintf(".backup-%s", timestamp))

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	for _, file := range files {
		if err := copyFile(file, filepath.Join(backupDir, filepath.Base(file))); err != nil {
			return "", fmt.Errorf("failed to backup %s: %w", file, err)
		}
	}

	return backupDir, nil
}

type BackupInfo struct {
	Path      string
	Timestamp string
	FileCount int
}

func ListBackups() ([]BackupInfo, error) {
	workflowDir := ".github/workflows"
	var backups []BackupInfo

	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), ".backup-") {
			backupPath := filepath.Join(workflowDir, entry.Name())
			if info, err := validateBackup(backupPath); err == nil {
				backups = append(backups, info)
			}
		}
	}

	return backups, nil
}

func validateBackup(backupPath string) (BackupInfo, error) {
	info := BackupInfo{Path: backupPath}

	parts := strings.Split(filepath.Base(backupPath), "-")
	if len(parts) >= 2 {
		info.Timestamp = strings.Join(parts[1:], "-")
	}

	entries, err := os.ReadDir(backupPath)
	if err != nil {
		return info, fmt.Errorf("failed to read backup directory: %w", err)
	}

	fileCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			ext := filepath.Ext(entry.Name())
			if ext == ".yml" || ext == ".yaml" {
				fileCount++
			}
		}
	}

	if fileCount == 0 {
		return info, fmt.Errorf("no workflow files found in backup")
	}

	info.FileCount = fileCount
	return info, nil
}

func RestoreBackup(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("no backup path provided")
	}

	if _, err := validateBackup(backupPath); err != nil {
		return fmt.Errorf("invalid backup: %w", err)
	}

	entries, err := os.ReadDir(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			ext := filepath.Ext(entry.Name())
			if ext == ".yml" || ext == ".yaml" {
				src := filepath.Join(backupPath, entry.Name())
				dst := filepath.Join(".github/workflows", entry.Name())
				if err := copyFile(src, dst); err != nil {
					return fmt.Errorf("failed to restore %s: %w", entry.Name(), err)
				}
			}
		}
	}

	return nil
}

func DeleteBackup(backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("no backup path provided")
	}

	if err := os.RemoveAll(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup directory: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := source.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := destination.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(destination, source)
	return err
}
