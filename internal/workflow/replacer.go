package workflow

import (
	"fmt"
	"os"
	"strings"
)

type Replacement struct {
	Action  ActionReference
	SHA     string
	Version string
}

func ReplaceActionsInFile(filePath string, replacements []Replacement) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	for _, repl := range replacements {
		for i, line := range lines {
			if strings.Contains(line, repl.Action.FullUses) {
				oldUses := fmt.Sprintf("%s/%s@%s", repl.Action.Owner, repl.Action.Repo, repl.Action.Ref)
				newUses := fmt.Sprintf("%s/%s@%s # %s", repl.Action.Owner, repl.Action.Repo, repl.SHA, repl.Version)
				lines[i] = strings.Replace(line, oldUses, newUses, 1)
			}
		}
	}

	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
