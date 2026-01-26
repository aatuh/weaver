package gitignore

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Rule represents a single .gitignore pattern.
type Rule struct {
	Raw      string
	Pattern  string
	Negate   bool
	DirOnly  bool
	Anchored bool
	HasSlash bool

	segments       []string
	segmentPattern string
}

// Matcher stores compiled rules.
type Matcher struct {
	rules []Rule
}

// NewMatcher creates a matcher from pre-parsed rules.
func NewMatcher(rules []Rule) *Matcher {
	return &Matcher{rules: rules}
}

// Rules returns the parsed rules in order.
func (m *Matcher) Rules() []Rule {
	return m.rules
}

// LoadFile loads a .gitignore file. If the file does not exist, an empty matcher is returned.
func LoadFile(path string) (*Matcher, error) {
	// #nosec G304 -- rule files are user-specified by design.
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewMatcher(nil), nil
		}
		return nil, err
	}
	defer file.Close()

	return Parse(file)
}

// Parse reads gitignore rules from a reader.
func Parse(r io.Reader) (*Matcher, error) {
	scanner := bufio.NewScanner(r)
	rules := make([]Rule, 0)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimRight(scanner.Text(), "\r")
		line = trimTrailingWhitespace(line)
		if line == "" {
			continue
		}
		if isComment(line) {
			continue
		}

		rule, err := parseRule(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
		if rule.Pattern == "" {
			continue
		}
		rules = append(rules, rule)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return NewMatcher(rules), nil
}

func isComment(line string) bool {
	return strings.HasPrefix(line, "#")
}

func trimTrailingWhitespace(line string) string {
	for len(line) > 0 {
		last := line[len(line)-1]
		if last != ' ' && last != '\t' {
			return line
		}
		if isEscaped(line, len(line)-1) {
			return line
		}
		line = line[:len(line)-1]
	}
	return line
}

func parseRule(line string) (Rule, error) {
	rule := Rule{Raw: line}

	if strings.HasPrefix(line, "!") {
		rule.Negate = true
		line = line[1:]
	}

	if strings.HasPrefix(line, "/") {
		rule.Anchored = true
		line = line[1:]
	}

	if len(line) > 0 && line[len(line)-1] == '/' && !isEscaped(line, len(line)-1) {
		rule.DirOnly = true
		line = line[:len(line)-1]
	}

	if line == "" {
		return rule, nil
	}

	rule.Pattern = line
	rule.HasSlash = rule.Anchored || strings.Contains(line, "/")

	if rule.HasSlash {
		segments := splitSegments(line)
		normalized := make([]string, 0, len(segments))
		for _, seg := range segments {
			seg = normalizeSegment(seg)
			if seg == "" {
				continue
			}
			if seg != "**" {
				if err := validateSegment(seg); err != nil {
					return rule, err
				}
			}
			normalized = append(normalized, seg)
		}
		rule.segments = normalized
		return rule, nil
	}

	segment := normalizeSegment(line)
	if err := validateSegment(segment); err != nil {
		return rule, err
	}
	rule.segmentPattern = segment
	return rule, nil
}

func normalizeSegment(segment string) string {
	if segment == "**" {
		return segment
	}
	for strings.Contains(segment, "**") {
		segment = strings.ReplaceAll(segment, "**", "*")
	}
	return segment
}

func validateSegment(segment string) error {
	_, err := path.Match(segment, "validate")
	if err != nil {
		return fmt.Errorf("invalid pattern %q: %w", segment, err)
	}
	return nil
}

func splitSegments(pattern string) []string {
	parts := strings.Split(pattern, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func splitPath(pathValue string) []string {
	parts := strings.Split(pathValue, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func isEscaped(line string, index int) bool {
	if index <= 0 {
		return false
	}
	count := 0
	for i := index - 1; i >= 0 && line[i] == '\\'; i-- {
		count++
	}
	return count%2 == 1
}

// Match reports whether the rule matches the provided path.
func (r Rule) Match(pathValue string, isDir bool) bool {
	if r.DirOnly && !isDir {
		return false
	}

	if r.HasSlash {
		return matchSegments(r.segments, splitPath(pathValue))
	}

	base := path.Base(pathValue)
	return matchSegment(r.segmentPattern, base)
}

// MatchDescendant reports whether the rule matches the path or any ancestor directory.
func (r Rule) MatchDescendant(pathValue string, isDir bool) bool {
	if !r.DirOnly {
		return r.Match(pathValue, isDir)
	}
	if r.Match(pathValue, isDir) {
		return true
	}

	current := pathValue
	if !isDir {
		current = path.Dir(pathValue)
	}
	for current != "." && current != "/" && current != "" {
		if r.Match(current, true) {
			return true
		}
		current = path.Dir(current)
	}
	return false
}

func matchSegment(pattern, name string) bool {
	matched, err := path.Match(pattern, name)
	if err != nil {
		return false
	}
	return matched
}

func matchSegments(patternSegments, pathSegments []string) bool {
	if len(patternSegments) == 0 {
		return len(pathSegments) == 0
	}

	pi := 0
	si := 0
	for pi < len(patternSegments) {
		pattern := patternSegments[pi]
		if pattern == "**" {
			if pi == len(patternSegments)-1 {
				return true
			}
			nextPattern := patternSegments[pi+1:]
			for i := si; i <= len(pathSegments); i++ {
				if matchSegments(nextPattern, pathSegments[i:]) {
					return true
				}
			}
			return false
		}

		if si >= len(pathSegments) {
			return false
		}
		if !matchSegment(pattern, pathSegments[si]) {
			return false
		}
		pi++
		si++
	}

	return si == len(pathSegments)
}

// RelativeGitPath converts an OS path to a slash-separated relative path.
func RelativeGitPath(root, pathValue string) (string, error) {
	rel, err := filepath.Rel(root, pathValue)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	return rel, nil
}
