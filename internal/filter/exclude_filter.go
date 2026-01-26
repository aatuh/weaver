package filter

// ExcludePathFilter prevents specific paths from being included or descended into.
type ExcludePathFilter struct {
	Inner    PathFilter
	Excluded map[string]struct{}
}

// NewExcludePathFilter wraps a filter with an exclusion list.
func NewExcludePathFilter(inner PathFilter, paths []string) PathFilter {
	if len(paths) == 0 {
		return inner
	}

	excluded := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		excluded[path] = struct{}{}
	}

	return ExcludePathFilter{Inner: inner, Excluded: excluded}
}

func (f ExcludePathFilter) Evaluate(path string, isDir bool) Decision {
	if _, ok := f.Excluded[path]; ok {
		return Decision{Include: false, Descend: false}
	}
	return f.Inner.Evaluate(path, isDir)
}
