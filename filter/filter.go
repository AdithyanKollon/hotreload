// Package filter provides path-based filtering to ignore files that
// should not trigger rebuilds (e.g. .git, node_modules, build artifacts).
package filter

import (
	"path/filepath"
	"strings"
)

// DefaultIgnorePatterns are the patterns ignored by default.
var DefaultIgnorePatterns = []string{
	".git",
	".svn",
	".hg",
	"node_modules",
	"vendor",
	".idea",
	".vscode",
	"__pycache__",
	"*.pyc",
	"*.pyo",
	"*.class",
	"*.o",
	"*.a",
	"*.so",
	"*.dylib",
	"*.dll",
	"*.exe",
	"*.test",
	"*.out",
	// Editor temp files
	"*.swp",
	"*.swo",
	"*~",
	".#*",
	"#*#",
	// OS junk
	".DS_Store",
	"Thumbs.db",
	// Common build output dirs
	"dist",
	"build",
	"bin",
	"tmp",
	"temp",
	".cache",
}

// Filter decides whether a given path should be ignored.
type Filter struct {
	patterns []string
}

// New creates a Filter that ignores the given glob patterns.
func New(patterns ...string) *Filter {
	return &Filter{patterns: patterns}
}

// ShouldIgnore returns true if the path matches any ignore pattern.
// It checks every component of the path so that e.g. "node_modules"
// matches "project/node_modules/foo.js".
func (f *Filter) ShouldIgnore(path string) bool {
	// Normalise separators
	path = filepath.ToSlash(path)

	parts := strings.Split(path, "/")
	for _, part := range parts {
		for _, pattern := range f.patterns {
			matched, err := filepath.Match(pattern, part)
			if err == nil && matched {
				return true
			}
		}
	}
	return false
}
