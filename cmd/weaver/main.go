package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aatu/weaver/internal/adapters/fs"
	"github.com/aatu/weaver/internal/app"
	"github.com/aatu/weaver/internal/filter"
	"github.com/aatu/weaver/internal/gitignore"
)

func main() {
	var (
		outFlag     = flag.String("out", "", "Output file path ('-' for stdout, defaults to stdout)")
		includeTree = flag.Bool("include-tree", false, "Include JSON file tree of included files")
	)
	var roots []string
	flag.Var(rootsFlag{Roots: &roots}, "root", "Root directory to scan (repeatable, defaults to current directory)")
	var ruleSpecs []ruleSpec
	flag.Var(ruleFlag{Mode: filter.ModeBlacklist, Specs: &ruleSpecs}, "blacklist", "Path to gitignore-style file to blacklist (repeatable)")
	flag.Var(ruleFlag{Mode: filter.ModeWhitelist, Specs: &ruleSpecs}, "whitelist", "Path to gitignore-style file to whitelist (repeatable)")

	flag.Usage = func() {
		usage(os.Stderr)
	}

	flag.Parse()

	if flag.NArg() > 0 {
		exitWithError(fmt.Errorf("unexpected arguments: %s", strings.Join(flag.Args(), ", ")))
	}

	if len(roots) == 0 {
		roots = []string{"."}
	}
	rootsAbs := make([]string, 0, len(roots))
	for _, root := range roots {
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			exitWithError(fmt.Errorf("resolve root %q: %w", root, err))
		}
		rootAbs = filepath.Clean(rootAbs)
		if err := validateRoot(rootAbs); err != nil {
			exitWithError(err)
		}
		rootsAbs = append(rootsAbs, rootAbs)
	}
	rootLabels := makeRootLabels(rootsAbs)

	outWriter, outAbs, err := prepareOutput(*outFlag)
	if err != nil {
		exitWithError(err)
	}
	defer outWriter.Close()

	excludedPaths := make([][]string, len(rootsAbs))
	if outAbs != "" {
		for i, root := range rootsAbs {
			if rel, ok := relativeIfWithin(root, outAbs); ok {
				excludedPaths[i] = append(excludedPaths[i], rel)
			}
		}
	}

	baseMode := filter.ModeBlacklist
	if len(ruleSpecs) > 0 {
		baseMode = ruleSpecs[0].Mode
	}
	filters := make([]filter.PathFilter, len(rootsAbs))
	for i, root := range rootsAbs {
		ruleSets, err := loadRuleSets(root, ruleSpecs)
		if err != nil {
			exitWithError(err)
		}
		baseFilter := filter.NewRuleSetFilter(ruleSets, baseMode)
		filters[i] = filter.NewExcludePathFilter(baseFilter, excludedPaths[i])
	}

	combiner := app.Combiner{FS: fs.OSFS{}}
	opts := app.Options{
		Roots:       rootsAbs,
		RootLabels:  rootLabels,
		Filters:     filters,
		IncludeTree: *includeTree,
		Output:      outWriter,
		ModeLabel:   formatRuleModes(ruleSpecs),
	}

	if err := combiner.Combine(context.Background(), opts); err != nil {
		exitWithError(err)
	}
}

func validateRoot(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("root is not a directory: %s", root)
	}
	return nil
}

func prepareOutput(outPath string) (io.WriteCloser, string, error) {
	if outPath == "" || outPath == "-" {
		return nopCloser{Writer: os.Stdout}, "", nil
	}

	outAbs, err := filepath.Abs(outPath)
	if err != nil {
		return nil, "", fmt.Errorf("resolve output: %w", err)
	}
	// #nosec G304 -- output path is user-provided by design.
	file, err := os.Create(outAbs)
	if err != nil {
		return nil, "", fmt.Errorf("create output: %w", err)
	}

	return file, outAbs, nil
}

func relativeIfWithin(rootAbs, targetAbs string) (string, bool) {
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return "", false
	}
	if rel == "." || rel == "" {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

type nopCloser struct {
	io.Writer
}

func (n nopCloser) Close() error {
	return nil
}

type ruleSpec struct {
	Mode filter.Mode
	Path string
}

type rootsFlag struct {
	Roots *[]string
}

func (f rootsFlag) String() string {
	if f.Roots == nil {
		return ""
	}
	return strings.Join(*f.Roots, ",")
}

func (f rootsFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("root path is required")
	}
	if f.Roots == nil {
		return fmt.Errorf("root destination is not configured")
	}
	*f.Roots = append(*f.Roots, value)
	return nil
}

type ruleFlag struct {
	Mode  filter.Mode
	Specs *[]ruleSpec
}

func (f ruleFlag) String() string {
	if f.Specs == nil {
		return ""
	}
	parts := make([]string, 0, len(*f.Specs))
	for _, spec := range *f.Specs {
		if spec.Mode == f.Mode {
			parts = append(parts, spec.Path)
		}
	}
	return strings.Join(parts, ",")
}

func (f ruleFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s rule path is required", f.Mode.String())
	}
	if f.Specs == nil {
		return fmt.Errorf("rule destination is not configured")
	}
	*f.Specs = append(*f.Specs, ruleSpec{Mode: f.Mode, Path: value})
	return nil
}

func resolveRulePath(rootAbs, rulePath string) string {
	if rulePath == "" {
		return rulePath
	}
	if filepath.IsAbs(rulePath) {
		return rulePath
	}
	absPath, err := filepath.Abs(rulePath)
	if err == nil {
		if _, statErr := os.Stat(absPath); statErr == nil || !os.IsNotExist(statErr) {
			return absPath
		}
	}
	return filepath.Join(rootAbs, rulePath)
}

func loadRuleSets(rootAbs string, ruleSpecs []ruleSpec) ([]filter.RuleSet, error) {
	ruleSets := make([]filter.RuleSet, 0, len(ruleSpecs))
	for _, spec := range ruleSpecs {
		rulePath := resolveRulePath(rootAbs, spec.Path)
		matcher, err := gitignore.LoadFile(rulePath)
		if err != nil {
			return nil, fmt.Errorf("load %s rules from %s: %w", spec.Mode.String(), rulePath, err)
		}
		ruleSets = append(ruleSets, filter.RuleSet{Mode: spec.Mode, Matcher: matcher})
	}
	return ruleSets, nil
}

func formatRuleModes(ruleSpecs []ruleSpec) string {
	if len(ruleSpecs) == 0 {
		return ""
	}
	if len(ruleSpecs) == 1 {
		return ruleSpecs[0].Mode.String()
	}
	parts := make([]string, 0, len(ruleSpecs))
	for _, spec := range ruleSpecs {
		parts = append(parts, spec.Mode.String())
	}
	return strings.Join(parts, " -> ")
}

func makeRootLabels(roots []string) []string {
	if len(roots) == 1 {
		return []string{defaultRootLabel(roots[0])}
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	labels := make([]string, len(roots))
	for i, root := range roots {
		label := ""
		if cwd != "" {
			rel, err := filepath.Rel(cwd, root)
			if err == nil {
				rel = filepath.Clean(rel)
				if rel != "." && rel != "" && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
					label = rel
				}
			}
		}
		if label == "" {
			label = defaultRootLabel(root)
		}
		labels[i] = filepath.ToSlash(label)
	}
	return makeUniqueLabels(labels)
}

func defaultRootLabel(root string) string {
	label := filepath.Base(root)
	if label == "." || label == "" || label == string(filepath.Separator) {
		label = "root"
	}
	return label
}

func makeUniqueLabels(labels []string) []string {
	used := make(map[string]bool, len(labels))
	out := make([]string, len(labels))
	for i, label := range labels {
		base := label
		if base == "" || base == "." || base == string(filepath.Separator) {
			base = fmt.Sprintf("root-%d", i+1)
		}
		candidate := base
		for suffix := 2; used[candidate]; suffix++ {
			candidate = fmt.Sprintf("%s-%d", base, suffix)
		}
		out[i] = candidate
		used[candidate] = true
	}
	return out
}

func usage(w io.Writer) {
	fmt.Fprintf(w, "Usage: %s [flags]\n\n", filepath.Base(os.Args[0]))
	fmt.Fprintln(w, "Weaver combines files from a directory into a single text file.")
	fmt.Fprintln(w, "Filtering is configured by one or more gitignore-style rule files.")
	fmt.Fprintln(w, "Rule files are evaluated in order; later matches override earlier ones.")
	fmt.Fprintln(w, "If no rule files are provided, all files are included.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	output := flag.CommandLine.Output()
	flag.CommandLine.SetOutput(w)
	flag.PrintDefaults()
	flag.CommandLine.SetOutput(output)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  weaver -root . -out combined.txt")
	fmt.Fprintln(w, "  weaver -root . -include-tree -out -")
	fmt.Fprintln(w, "  weaver -root ./api -root ./web -out -")
	fmt.Fprintln(w, "  weaver -blacklist .gitignore -out -")
}
