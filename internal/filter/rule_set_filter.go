package filter

import "github.com/aatuh/weaver/internal/gitignore"

// RuleSet bundles a matcher with its evaluation mode.
type RuleSet struct {
	Mode    Mode
	Matcher *gitignore.Matcher
}

// RuleSetFilter applies multiple rule sets in order, letting later matches override earlier ones.
type RuleSetFilter struct {
	BaseMode Mode
	RuleSets []RuleSet
}

// NewRuleSetFilter returns a filter that evaluates rule sets in order.
func NewRuleSetFilter(ruleSets []RuleSet, baseMode Mode) PathFilter {
	if len(ruleSets) == 0 {
		return GitIgnoreFilter{Mode: baseMode}
	}
	return RuleSetFilter{BaseMode: baseMode, RuleSets: ruleSets}
}

func (f RuleSetFilter) Evaluate(path string, isDir bool) Decision {
	baseDecision := decisionForMatch(f.BaseMode, false, false)
	if len(f.RuleSets) == 0 {
		return baseDecision
	}

	matchedAny := false
	decision := baseDecision
	for _, rules := range f.RuleSets {
		matched, negated := matchRules(rules.Mode, rules.Matcher, path, isDir)
		if !matched {
			continue
		}
		matchedAny = true
		decision = decisionForMatch(rules.Mode, matched, negated)
	}

	if matchedAny {
		return decision
	}
	return baseDecision
}
