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

	if err := os.WriteFile(filepath.Join(rootA, "a.txt"), []byte("A"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rootB, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootB, "sub", "b.txt"), []byte("B"), 0o600); err != nil {
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
		MaxDepth:   -1,
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
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("A"), 0o600); err != nil {
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
		MaxDepth:           -1,
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

func TestCombinerMaxDepthLimitsNestedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "root.txt"), []byte("root"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "deep.txt"), []byte("deep"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var buf bytes.Buffer
	combiner := Combiner{
		FS:    fs.OSFS{},
		Clock: func() time.Time { return time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC) },
	}
	allowAll := filter.GitIgnoreFilter{Mode: filter.ModeBlacklist}
	opts := Options{
		Roots:      []string{root},
		RootLabels: []string{"root"},
		Filters:    []filter.PathFilter{allowAll},
		MaxDepth:   0,
		Output:     &buf,
	}

	if err := combiner.Combine(context.Background(), opts); err != nil {
		t.Fatalf("combine: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "--- BEGIN FILE: root.txt ---") {
		t.Fatalf("expected root-level file, got output:\n%s", output)
	}
	if strings.Contains(output, "nested/deep.txt") {
		t.Fatalf("unexpected nested file in output:\n%s", output)
	}
}

func TestCombinerSkipContentsOmitsFileSections(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("A"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var buf bytes.Buffer
	combiner := Combiner{
		FS:    fs.OSFS{},
		Clock: func() time.Time { return time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC) },
	}
	allowAll := filter.GitIgnoreFilter{Mode: filter.ModeBlacklist}
	opts := Options{
		Roots:        []string{root},
		RootLabels:   []string{"root"},
		Filters:      []filter.PathFilter{allowAll},
		IncludeTree:  true,
		MaxDepth:     -1,
		SkipContents: true,
		Output:       &buf,
	}

	if err := combiner.Combine(context.Background(), opts); err != nil {
		t.Fatalf("combine: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "--- BEGIN FILE TREE (JSON) ---") {
		t.Fatalf("expected tree output, got:\n%s", output)
	}
	if strings.Contains(output, "--- BEGIN FILE:") {
		t.Fatalf("did not expect file sections when skip-contents is enabled, got:\n%s", output)
	}
}

func TestCombinerSkipBinaryWritesPlaceholder(t *testing.T) {
	root := t.TempDir()
	binary := []byte{0x00, 0x01, 0x02, 0x03}
	if err := os.WriteFile(filepath.Join(root, "bin.dat"), binary, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var buf bytes.Buffer
	combiner := Combiner{
		FS:    fs.OSFS{},
		Clock: func() time.Time { return time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC) },
	}
	allowAll := filter.GitIgnoreFilter{Mode: filter.ModeBlacklist}
	opts := Options{
		Roots:      []string{root},
		RootLabels: []string{"root"},
		Filters:    []filter.PathFilter{allowAll},
		MaxDepth:   -1,
		SkipBinary: true,
		Output:     &buf,
	}

	if err := combiner.Combine(context.Background(), opts); err != nil {
		t.Fatalf("combine: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "--- BEGIN FILE: bin.dat ---") {
		t.Fatalf("expected binary file section, got:\n%s", output)
	}
	if !strings.Contains(output, "[binary content omitted]") {
		t.Fatalf("expected binary placeholder, got:\n%s", output)
	}
}
