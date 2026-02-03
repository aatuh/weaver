package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aatuh/weaver/internal/adapters/fs"
	"github.com/aatuh/weaver/internal/filter"
)

func TestCombinerMultipleRootsPrefixesDisplayPaths(t *testing.T) {
	rootA := t.TempDir()
	rootB := t.TempDir()

	if err := os.WriteFile(filepath.Join(rootA, "a.txt"), []byte("A"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rootB, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootB, "sub", "b.txt"), []byte("B"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var buf bytes.Buffer
	combiner := Combiner{
		FS:    fs.OSFS{},
		Clock: func() time.Time { return time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC) },
	}
	allowAll := filter.GitIgnoreFilter{Mode: filter.ModeBlacklist}
	opts := Options{
		Roots:      []string{rootA, rootB},
		RootLabels: []string{"root-a", "root-b"},
		Filters:    []filter.PathFilter{allowAll, allowAll},
		Output:     &buf,
	}

	if err := combiner.Combine(context.Background(), opts); err != nil {
		t.Fatalf("combine: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "# Roots:\n") {
		t.Fatalf("expected roots header, got output:\n%s", output)
	}
	if !strings.Contains(output, "--- BEGIN FILE: root-a/a.txt ---") {
		t.Fatalf("expected root-a prefixed file, got output:\n%s", output)
	}
	if !strings.Contains(output, "--- BEGIN FILE: root-b/sub/b.txt ---") {
		t.Fatalf("expected root-b prefixed file, got output:\n%s", output)
	}
}

func TestCombinerCompactTreeIsSingleLine(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("A"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var buf bytes.Buffer
	combiner := Combiner{
		FS:    fs.OSFS{},
		Clock: func() time.Time { return time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC) },
	}
	allowAll := filter.GitIgnoreFilter{Mode: filter.ModeBlacklist}
	opts := Options{
		Roots:              []string{root},
		RootLabels:         []string{"root"},
		Filters:            []filter.PathFilter{allowAll},
		IncludeTreeCompact: true,
		Output:             &buf,
	}

	if err := combiner.Combine(context.Background(), opts); err != nil {
		t.Fatalf("combine: %v", err)
	}

	output := buf.String()
	start := strings.Index(output, "--- BEGIN FILE TREE (JSON, COMPACT) ---\n")
	if start == -1 {
		t.Fatalf("missing compact tree header, got output:\n%s", output)
	}
	segment := output[start:]
	parts := strings.SplitN(segment, "\n", 3)
	if len(parts) < 3 {
		t.Fatalf("unexpected compact tree section, got output:\n%s", output)
	}
	payload := parts[1]
	if strings.Contains(payload, "\n") {
		t.Fatalf("expected compact tree to be single line, got:\n%s", payload)
	}
}
