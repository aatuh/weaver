package filter

import (
	"strings"
	"testing"

	"github.com/aatu/weaver/internal/gitignore"
)

func mustMatcher(t *testing.T, content string) *gitignore.Matcher {
	t.Helper()
	matcher, err := gitignore.Parse(strings.NewReader(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return matcher
}

func TestBlacklistNegation(t *testing.T) {
	matcher := mustMatcher(t, "*.log\n!important.log\n")
	filter := GitIgnoreFilter{Mode: ModeBlacklist, Matcher: matcher}

	if got := filter.Evaluate("debug.log", false).Include; got {
		t.Fatalf("expected debug.log to be excluded")
	}
	if got := filter.Evaluate("important.log", false).Include; !got {
		t.Fatalf("expected important.log to be included")
	}
	if got := filter.Evaluate("logs/trace.log", false).Include; got {
		t.Fatalf("expected logs/trace.log to be excluded")
	}
}

func TestBlacklistAnchoredAndSlash(t *testing.T) {
	matcher := mustMatcher(t, "doc/*.txt\n/TODO\n")
	filter := GitIgnoreFilter{Mode: ModeBlacklist, Matcher: matcher}

	if got := filter.Evaluate("doc/readme.txt", false).Include; got {
		t.Fatalf("expected doc/readme.txt to be excluded")
	}
	if got := filter.Evaluate("doc/sub/readme.txt", false).Include; !got {
		t.Fatalf("expected doc/sub/readme.txt to be included")
	}
	if got := filter.Evaluate("TODO", false).Include; got {
		t.Fatalf("expected TODO to be excluded")
	}
	if got := filter.Evaluate("src/TODO", false).Include; !got {
		t.Fatalf("expected src/TODO to be included")
	}
}

func TestBlacklistGlobstar(t *testing.T) {
	matcher := mustMatcher(t, "a/**/b\n")
	filter := GitIgnoreFilter{Mode: ModeBlacklist, Matcher: matcher}

	if got := filter.Evaluate("a/b", false).Include; got {
		t.Fatalf("expected a/b to be excluded")
	}
	if got := filter.Evaluate("a/x/b", false).Include; got {
		t.Fatalf("expected a/x/b to be excluded")
	}
	if got := filter.Evaluate("a/x/y/b", false).Include; got {
		t.Fatalf("expected a/x/y/b to be excluded")
	}
	if got := filter.Evaluate("a/b/c", false).Include; !got {
		t.Fatalf("expected a/b/c to be included")
	}
}

func TestWhitelistDirOnly(t *testing.T) {
	matcher := mustMatcher(t, "assets/\n!assets/secret.txt\n")
	filter := GitIgnoreFilter{Mode: ModeWhitelist, Matcher: matcher}

	if got := filter.Evaluate("assets/img.png", false).Include; !got {
		t.Fatalf("expected assets/img.png to be included")
	}
	if got := filter.Evaluate("assets/secret.txt", false).Include; got {
		t.Fatalf("expected assets/secret.txt to be excluded")
	}
	if got := filter.Evaluate("other.txt", false).Include; got {
		t.Fatalf("expected other.txt to be excluded")
	}
}

func TestWhitelistBasenameOnly(t *testing.T) {
	matcher := mustMatcher(t, "docs\n")
	filter := GitIgnoreFilter{Mode: ModeWhitelist, Matcher: matcher}

	if got := filter.Evaluate("docs", true).Include; !got {
		t.Fatalf("expected docs directory to be included")
	}
	if got := filter.Evaluate("docs/readme.md", false).Include; got {
		t.Fatalf("expected docs/readme.md to be excluded without explicit dir pattern")
	}
}
