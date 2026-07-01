package doclint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Result struct {
	Errors   []string
	Warnings []string
}

func Check(root string) Result {
	var result Result
	required := []string{
		"AGENTS.md",
		"ARCHITECTURE.md",
		"README.md",
		"docs/PLANS.md",
		"docs/QUALITY_SCORE.md",
		"docs/RELIABILITY.md",
		"docs/SECURITY.md",
		"docs/product-specs/index.md",
		"docs/exec-plans/tech-debt-tracker.md",
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("required file missing: %s", rel))
		}
	}

	checkAgentsLength(root, &result)
	checkActivePlans(root, &result)
	return result
}

func checkAgentsLength(root string, result *Result) {
	path := filepath.Join(root, "AGENTS.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Count(string(data), "\n") + 1
	if lines > 120 {
		result.Errors = append(result.Errors, fmt.Sprintf("AGENTS.md has %d lines; keep it under 120 and move details into docs/", lines))
	}
	for _, mustMention := range []string{"ARCHITECTURE.md", "docs/exec-plans/active", "go test ./..."} {
		if !strings.Contains(string(data), mustMention) {
			result.Errors = append(result.Errors, fmt.Sprintf("AGENTS.md should mention %s", mustMention))
		}
	}
}

func checkActivePlans(root string, result *Result) {
	dir := filepath.Join(root, "docs", "exec-plans", "active")
	entries, err := os.ReadDir(dir)
	if err != nil {
		result.Errors = append(result.Errors, "active plan directory missing: docs/exec-plans/active")
		return
	}
	if len(entries) == 0 {
		result.Warnings = append(result.Warnings, "no active execution plans found")
		return
	}
	requiredSections := []string{"## Status", "## Goal", "## Scope", "## Non-goals", "## Acceptance criteria", "## Verification", "## Decision log", "## Next actions"}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		rel := filepath.Join("docs", "exec-plans", "active", entry.Name())
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("cannot read active plan %s: %v", rel, err))
			continue
		}
		text := string(data)
		for _, section := range requiredSections {
			if !strings.Contains(text, section) {
				result.Errors = append(result.Errors, fmt.Sprintf("active plan %s missing section %s", rel, section))
			}
		}
	}
}
