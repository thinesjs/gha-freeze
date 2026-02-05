package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FindWorkflowFiles() ([]string, error) {
	workflowDir := ".github/workflows"

	if _, err := os.Stat(workflowDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("workflows directory not found: %s", workflowDir)
	}

	var workflows []string

	err := filepath.Walk(workflowDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".backup-") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		if ext == ".yml" || ext == ".yaml" {
			workflows = append(workflows, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan workflows directory: %w", err)
	}

	return workflows, nil
}
