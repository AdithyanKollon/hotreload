package filter_test

import (
	"testing"

	"github.com/AdithyanKollon/hotreload/filter"
)

func TestDefaultIgnorePatterns(t *testing.T) {
	f := filter.New(filter.DefaultIgnorePatterns...)

	cases := []struct {
		path   string
		ignore bool
	}{
		{"project/.git/HEAD", true},
		{"project/.git", true},
		{"project/node_modules/lodash/index.js", true},
		{"project/vendor/pkg/file.go", true},
		{"project/main.go", false},
		{"project/internal/handler.go", false},
		{"project/cmd/server/main.go", false},
		{"project/file.swp", true},
		{"project/file~", true},
		{"project/.DS_Store", true},
		{"project/bin/server", true},
		{"project/dist/output.js", true},
		{"project/src/app.go", false},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got := f.ShouldIgnore(tc.path)
			if got != tc.ignore {
				t.Errorf("ShouldIgnore(%q) = %v, want %v", tc.path, got, tc.ignore)
			}
		})
	}
}

func TestCustomPatterns(t *testing.T) {
	f := filter.New("*.log", "tmp")

	if !f.ShouldIgnore("app.log") {
		t.Error("expected app.log to be ignored")
	}
	if !f.ShouldIgnore("project/tmp/file.go") {
		t.Error("expected project/tmp/file.go to be ignored")
	}
	if f.ShouldIgnore("main.go") {
		t.Error("expected main.go not to be ignored")
	}
}
