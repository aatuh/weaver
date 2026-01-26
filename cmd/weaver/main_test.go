package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRulePathPrefersCwd(t *testing.T) {
	base := t.TempDir()
	rootAbs := filepath.Join(base, ".test")
	if err := os.MkdirAll(rootAbs, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ruleAbs := filepath.Join(rootAbs, ".gitignore")
	if err := os.WriteFile(ruleAbs, []byte(""), 0o644); err != nil {
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
	if err := os.WriteFile(ruleAbs, []byte(""), 0o644); err != nil {
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
