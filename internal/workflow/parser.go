package workflow

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type ActionReference struct {
	Owner    string
	Repo     string
	Ref      string
	Line     int
	FilePath string
	FullUses string
	IsPinned bool
}

var actionRegex = regexp.MustCompile(`^([^/]+)/([^@]+)@(.+)$`)
var shaRegex = regexp.MustCompile(`^[a-f0-9]{40}$`)

func ParseWorkflowFile(filePath string) ([]ActionReference, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var workflow map[string]interface{}
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	var actions []ActionReference
	lines := strings.Split(string(content), "\n")

	jobs, ok := workflow["jobs"].(map[string]interface{})
	if !ok {
		return actions, nil
	}

	for _, job := range jobs {
		jobMap, ok := job.(map[string]interface{})
		if !ok {
			continue
		}

		steps, ok := jobMap["steps"].([]interface{})
		if !ok {
			continue
		}

		for _, step := range steps {
			stepMap, ok := step.(map[string]interface{})
			if !ok {
				continue
			}

			uses, ok := stepMap["uses"].(string)
			if !ok {
				continue
			}

			lineNum := findLineNumber(lines, uses)
			action := parseActionString(uses, filePath, lineNum)
			if action != nil {
				actions = append(actions, *action)
			}
		}
	}

	return actions, nil
}

func parseActionString(uses, filePath string, lineNum int) *ActionReference {
	uses = strings.TrimSpace(uses)

	matches := actionRegex.FindStringSubmatch(uses)
	if matches == nil {
		return nil
	}

	ref := matches[3]
	isPinned := shaRegex.MatchString(ref)

	return &ActionReference{
		Owner:    matches[1],
		Repo:     matches[2],
		Ref:      ref,
		Line:     lineNum,
		FilePath: filePath,
		FullUses: uses,
		IsPinned: isPinned,
	}
}

func findLineNumber(lines []string, searchText string) int {
	for i, line := range lines {
		if strings.Contains(line, "uses:") && strings.Contains(line, searchText) {
			return i + 1
		}
	}
	return 0
}
