package filter

import "github.com/aatuh/weaver/internal/gitignore"

// GitIgnoreFilter evaluates paths using gitignore rules.
type GitIgnoreFilter struct {
	Mode    Mode
	Matcher *gitignore.Matcher
}

func (f GitIgnoreFilter) Evaluate(path string, isDir bool) Decision {
	matched, negated := matchRules(f.Mode, f.Matcher, path, isDir)
	return decisionForMatch(f.Mode, matched, negated)
}

func matchRules(mode Mode, matcher *gitignore.Matcher, path string, isDir bool) (bool, bool) {
	if matcher == nil {
		return false, false
	}
	rules := matcher.Rules()
	if len(rules) == 0 {
		return false, false
	}

	matched := false
	negated := false
	for _, rule := range rules {
		var ruleMatches bool
		if mode == ModeWhitelist && rule.DirOnly {
			ruleMatches = rule.MatchDescendant(path, isDir)
		} else {
			ruleMatches = rule.Match(path, isDir)
		}
		if ruleMatches {
			matched = true
			negated = rule.Negate
		}
	}

	return matched, negated
}

func decisionForMatch(mode Mode, matched, negated bool) Decision {
	switch mode {
	case ModeWhitelist:
		include := matched && !negated
		return Decision{Include: include, Descend: true}
	case ModeBlacklist:
		ignored := false
		if matched {
			ignored = !negated
		}
		include := !ignored
		return Decision{Include: include, Descend: include}
	}

	return Decision{Include: true, Descend: true}
}
