package filter

import (
	"strings"
	"testing"

	"github.com/aatuh/weaver/internal/gitignore"
)

func mustMatcherForRuleSet(t *testing.T, content string) *gitignore.Matcher {
	t.Helper()
	matcher, err := gitignore.Parse(strings.NewReader(content))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return matcher
}

func TestRuleSetFilterOverrideOrder(t *testing.T) {
	blacklist := mustMatcherForRuleSet(t, "*.log\n")
	whitelist := mustMatcherForRuleSet(t, "important.log\n")

	filter := RuleSetFilter{
		BaseMode: ModeBlacklist,
		RuleSets: []RuleSet{
			{Mode: ModeBlacklist, Matcher: blacklist},
			{Mode: ModeWhitelist, Matcher: whitelist},
		},
	}

	if got := filter.Evaluate("debug.log", false).Include; got {
		t.Fatalf("expected debug.log to be excluded by blacklist")
	}
	if got := filter.Evaluate("important.log", false).Include; !got {
		t.Fatalf("expected important.log to be included by whitelist override")
	}
	if got := filter.Evaluate("notes.txt", false).Include; !got {
		t.Fatalf("expected notes.txt to be included by default")
	}
}

func TestRuleSetFilterBaselineWhitelist(t *testing.T) {
	whitelist := mustMatcherForRuleSet(t, "docs/\n")
	blacklist := mustMatcherForRuleSet(t, "docs/secret.txt\n")

	filter := RuleSetFilter{
		BaseMode: ModeWhitelist,
		RuleSets: []RuleSet{
			{Mode: ModeWhitelist, Matcher: whitelist},
			{Mode: ModeBlacklist, Matcher: blacklist},
		},
	}

	if got := filter.Evaluate("docs/readme.md", false).Include; !got {
		t.Fatalf("expected docs/readme.md to be included by whitelist")
	}
	if got := filter.Evaluate("docs/secret.txt", false).Include; got {
		t.Fatalf("expected docs/secret.txt to be excluded by blacklist override")
	}
	if got := filter.Evaluate("other.txt", false).Include; got {
		t.Fatalf("expected other.txt to be excluded by whitelist baseline")
	}
}
