package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aatuh/weaver/internal/filter"
)

func TestResolveRulePathPrefersCwd(t *testing.T) {
	base := t.TempDir()
	rootAbs := filepath.Join(base, ".test")
	if err := os.MkdirAll(rootAbs, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ruleAbs := filepath.Join(rootAbs, ".gitignore")
	if err := os.WriteFile(ruleAbs, []byte(""), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()
	if err := os.Chdir(base); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	got := resolveRulePath(rootAbs, ".test/.gitignore")
	if got != ruleAbs {
		t.Fatalf("expected %s, got %s", ruleAbs, got)
	}
}

func TestResolveRulePathFallbacksToRoot(t *testing.T) {
	base := t.TempDir()
	rootAbs := filepath.Join(base, "root")
	if err := os.MkdirAll(rootAbs, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ruleAbs := filepath.Join(rootAbs, "rules.ignore")
	if err := os.WriteFile(ruleAbs, []byte(""), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()
	if err := os.Chdir(base); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	got := resolveRulePath(rootAbs, "rules.ignore")
	if got != ruleAbs {
		t.Fatalf("expected %s, got %s", ruleAbs, got)
	}
}

func TestLoadRuleSetsInlinePattern(t *testing.T) {
	rootAbs := t.TempDir()
	ruleSets, err := loadRuleSets(rootAbs, []ruleSpec{
		{Mode: filter.ModeBlacklist, Pattern: "*.log"},
	})
	if err != nil {
		t.Fatalf("load rule sets: %v", err)
	}

	pathFilter := filter.NewRuleSetFilter(ruleSets, filter.ModeBlacklist)
	if decision := pathFilter.Evaluate("debug.log", false); decision.Include {
		t.Fatalf("expected .log to be excluded by inline pattern")
	}
	if decision := pathFilter.Evaluate("debug.txt", false); !decision.Include {
		t.Fatalf("expected .txt to be included with inline pattern")
	}
}
