package sourcemap_test

import (
	"github.com/go-sourcemap/sourcemap"
	"testing"
)

func TestSourceDoesNotCrossGeneratedLines(t *testing.T) {
	// Mappings at line 1, column 0 and line 3, column 5.
	consumer, err := sourcemap.Parse("", []byte(`{"version":3,"sources":["input.js"],"names":[],"mappings":"AAAA;;KACA"}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		line, column int
		want         bool
	}{
		{1, 0, true}, {1, 100, true}, {2, 0, false}, {2, 100, false},
		{3, 0, false}, {3, 4, false}, {3, 5, true}, {3, 100, true}, {4, 0, false},
	} {
		_, _, _, _, got := consumer.Source(tc.line, tc.column)
		if got != tc.want {
			t.Errorf("Source(%d, %d) matched=%v, want %v", tc.line, tc.column, got, tc.want)
		}
	}
}
