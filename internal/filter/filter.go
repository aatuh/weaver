package filter

// Decision describes whether a path should be included and whether to descend into it.
type Decision struct {
	Include bool
	Descend bool
}

// PathFilter decides whether a path should be included and whether to descend into directories.
type PathFilter interface {
	Evaluate(path string, isDir bool) Decision
}

// Mode defines how ignore rules are applied.
type Mode int

const (
	ModeBlacklist Mode = iota
	ModeWhitelist
)

func (m Mode) String() string {
	switch m {
	case ModeBlacklist:
		return "blacklist"
	case ModeWhitelist:
		return "whitelist"
	default:
		return "unknown"
	}
}
