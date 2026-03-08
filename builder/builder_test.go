package builder_test

import (
	"testing"

	"github.com/AdithyanKollon/hotreload/builder"
)

func TestSplitCommand(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{
			"go build -o ./bin/server ./cmd/server",
			[]string{"go", "build", "-o", "./bin/server", "./cmd/server"},
		},
		{
			`go build -ldflags "-s -w" -o ./bin/server`,
			[]string{"go", "build", "-ldflags", "-s -w", "-o", "./bin/server"},
		},
		{
			"./bin/server",
			[]string{"./bin/server"},
		},
		{
			"",
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := builder.SplitCommand(tc.input)
			if len(got) != len(tc.expected) {
				t.Errorf("SplitCommand(%q) = %v, want %v", tc.input, got, tc.expected)
				return
			}
			for i := range got {
				if got[i] != tc.expected[i] {
					t.Errorf("SplitCommand(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.expected[i])
				}
			}
		})
	}
}
