package doclint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckReportsMissingRequiredFiles(t *testing.T) {
	dir := t.TempDir()
	result := Check(dir)
	if len(result.Errors) == 0 {
		t.Fatal("expected missing file errors")
	}
}

func TestCheckAcceptsMinimalHarness(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, dir, "AGENTS.md", "# AGENTS.md\nRead ARCHITECTURE.md and docs/exec-plans/active. Run go test ./...\n")
	mustWrite(t, dir, "ARCHITECTURE.md", "# Architecture\n")
	mustWrite(t, dir, "README.md", "# Readme\n")
	mustWrite(t, dir, "docs/PLANS.md", "# Plans\n")
	mustWrite(t, dir, "docs/QUALITY_SCORE.md", "# Quality\n")
	mustWrite(t, dir, "docs/RELIABILITY.md", "# Reliability\n")
	mustWrite(t, dir, "docs/SECURITY.md", "# Security\n")
	mustWrite(t, dir, "docs/product-specs/index.md", "# Product\n")
	mustWrite(t, dir, "docs/exec-plans/tech-debt-tracker.md", "# Debt\n")
	mustWrite(t, dir, "docs/exec-plans/active/0001.md", "# Plan\n\n## Status\n\n## Goal\n\n## Scope\n\n## Non-goals\n\n## Acceptance criteria\n\n## Verification\n\n## Decision log\n\n## Next actions\n")

	result := Check(dir)
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
}

func mustWrite(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
