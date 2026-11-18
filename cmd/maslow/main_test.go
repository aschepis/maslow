package main

import "testing"

func TestExitCode(t *testing.T) {
	cases := []struct {
		verdict string
		strict  bool
		want    int
	}{
		{"pass", false, 0},
		{"pass", true, 0},
		{"needs_review", false, 0},
		{"needs_review", true, 1},
		{"inconclusive", false, 1},
		{"inconclusive", true, 1},
		{"fail", false, 1},
		{"fail", true, 1},
	}
	for _, c := range cases {
		if got := exitCode(c.verdict, c.strict); got != c.want {
			t.Errorf("exitCode(%q, strict=%v) = %d, want %d", c.verdict, c.strict, got, c.want)
		}
	}
}
