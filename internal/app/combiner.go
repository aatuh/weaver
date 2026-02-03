package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aatuh/weaver/internal/filter"
	"github.com/aatuh/weaver/internal/tree"
)

// FileSystem provides file walking and reading.
type FileSystem interface {
	WalkDir(root string, fn fs.WalkDirFunc) error
	ReadFile(path string) ([]byte, error)
}

// Options configure the combine operation.
type Options struct {
	Roots              []string
	RootLabels         []string
	Filters            []filter.PathFilter
	IncludeTree        bool
	IncludeTreeCompact bool
	Output             io.Writer
	ModeLabel          string
}

// Combiner orchestrates collecting and writing combined files.
type Combiner struct {
	FS    FileSystem
	Clock func() time.Time
}

// Combine generates a combined file from the root directory.
func (c Combiner) Combine(ctx context.Context, opts Options) error {
	if len(opts.Roots) == 0 {
		return fmt.Errorf("root path is required")
	}
	if len(opts.Filters) != len(opts.Roots) {
		return fmt.Errorf("path filter is required")
	}
	if len(opts.RootLabels) != len(opts.Roots) {
		return fmt.Errorf("root labels are required")
	}
	if opts.Output == nil {
		return fmt.Errorf("output writer is required")
	}
	if c.FS == nil {
		return fmt.Errorf("filesystem adapter is required")
	}
	if c.Clock == nil {
		c.Clock = time.Now
	}

	type fileEntry struct {
		root    string
		rel     string
		display string
	}
	entries := make([]fileEntry, 0)
	for i, root := range opts.Roots {
		files, err := c.collectFiles(ctx, root, opts.Filters[i])
		if err != nil {
			return err
		}
		label := opts.RootLabels[i]
		for _, rel := range files {
			display := rel
			if len(opts.Roots) > 1 {
				display = path.Join(label, rel)
			}
			entries = append(entries, fileEntry{root: root, rel: rel, display: display})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].display < entries[j].display
	})

	writer := bufio.NewWriter(opts.Output)

	if err := c.writeHeader(writer, opts, len(entries)); err != nil {
		return err
	}

	if opts.IncludeTree || opts.IncludeTreeCompact {
		rootName := "roots"
		if len(opts.Roots) == 1 {
			rootName = opts.RootLabels[0]
		}
		paths := make([]string, len(entries))
		for i, entry := range entries {
			paths[i] = entry.display
		}
		treeNode := tree.Build(rootName, paths)

		if opts.IncludeTree {
			payload, err := json.MarshalIndent(treeNode, "", "  ")
			if err != nil {
				return fmt.Errorf("build tree: %w", err)
			}
			if err := writeString(writer, "--- BEGIN FILE TREE (JSON) ---\n"); err != nil {
				return err
			}
			if _, err := writer.Write(payload); err != nil {
				return err
			}
			if err := writeString(writer, "\n--- END FILE TREE ---\n\n"); err != nil {
				return err
			}
		}

		if opts.IncludeTreeCompact {
			payload, err := json.Marshal(treeNode)
			if err != nil {
				return fmt.Errorf("build compact tree: %w", err)
			}
			if err := writeString(writer, "--- BEGIN FILE TREE (JSON, COMPACT) ---\n"); err != nil {
				return err
			}
			if _, err := writer.Write(payload); err != nil {
				return err
			}
			if err := writeString(writer, "\n--- END FILE TREE (JSON, COMPACT) ---\n\n"); err != nil {
				return err
			}
		}
	}

	for _, entry := range entries {
		if err := writeString(writer, fmt.Sprintf("--- BEGIN FILE: %s ---\n", entry.display)); err != nil {
			return err
		}

		fullPath := filepath.Join(entry.root, filepath.FromSlash(entry.rel))
		data, err := c.FS.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.display, err)
		}
		if _, err := writer.Write(data); err != nil {
			return err
		}
		if len(data) == 0 || data[len(data)-1] != '\n' {
			if err := writeString(writer, "\n"); err != nil {
				return err
			}
		}
		if err := writeString(writer, fmt.Sprintf("--- END FILE: %s ---\n\n", entry.display)); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func (c Combiner) collectFiles(ctx context.Context, root string, pathFilter filter.PathFilter) ([]string, error) {
	files := make([]string, 0)

	err := c.FS.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx != nil {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." || strings.HasPrefix(rel, "../") || rel == ".." {
			return nil
		}

		decision := pathFilter.Evaluate(rel, entry.IsDir())
		if entry.IsDir() {
			if !decision.Descend {
				return fs.SkipDir
			}
			return nil
		}
		if decision.Include {
			files = append(files, rel)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}

func (c Combiner) writeHeader(writer *bufio.Writer, opts Options, count int) error {
	timestamp := c.Clock().UTC().Format(time.RFC3339)

	if err := writeString(writer, "# Weaver Combined File\n"); err != nil {
		return err
	}
	if len(opts.Roots) == 1 {
		if err := writeString(writer, fmt.Sprintf("# Root: %s\n", opts.Roots[0])); err != nil {
			return err
		}
	} else {
		if err := writeString(writer, "# Roots:\n"); err != nil {
			return err
		}
		for _, root := range opts.Roots {
			if err := writeString(writer, fmt.Sprintf("# - %s\n", root)); err != nil {
				return err
			}
		}
	}
	if opts.ModeLabel != "" {
		if err := writeString(writer, fmt.Sprintf("# Mode: %s\n", opts.ModeLabel)); err != nil {
			return err
		}
	}
	if err := writeString(writer, fmt.Sprintf("# Files: %d\n", count)); err != nil {
		return err
	}
	if err := writeString(writer, fmt.Sprintf("# Generated: %s\n\n", timestamp)); err != nil {
		return err
	}
	return nil
}

func writeString(writer *bufio.Writer, value string) error {
	_, err := writer.WriteString(value)
	return err
}
