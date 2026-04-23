package analysis

import "testing"

func TestToCamelot(t *testing.T) {
	cases := []struct {
		key, scale, want string
	}{
		// Major -> B side
		{"C", "major", "8B"},
		{"G", "major", "9B"},
		{"D", "major", "10B"},
		{"A", "major", "11B"},
		{"E", "major", "12B"},
		{"B", "major", "1B"},
		{"F#", "major", "2B"},
		{"Db", "major", "3B"},
		{"Ab", "major", "4B"},
		{"Eb", "major", "5B"},
		{"Bb", "major", "6B"},
		{"F", "major", "7B"},
		// Minor -> A side
		{"A", "minor", "8A"},
		{"E", "minor", "9A"},
		{"B", "minor", "10A"},
		{"F#", "minor", "11A"},
		{"C#", "minor", "12A"},
		{"G#", "minor", "1A"},
		{"Eb", "minor", "2A"},
		{"Bb", "minor", "3A"},
		{"F", "minor", "4A"},
		{"C", "minor", "5A"},
		{"G", "minor", "6A"},
		{"D", "minor", "7A"},
		// Enharmonic equivalents accepted
		{"Gb", "major", "2B"}, // same as F#
		{"C#", "major", "3B"}, // same as Db
		{"D#", "minor", "2A"}, // same as Eb
	}
	for _, tc := range cases {
		got := ToCamelot(tc.key, tc.scale)
		if got != tc.want {
			t.Errorf("ToCamelot(%q, %q) = %q, want %q", tc.key, tc.scale, got, tc.want)
		}
	}
}

func TestToCamelot_Invalid(t *testing.T) {
	cases := []struct{ key, scale string }{
		{"", "major"},
		{"H", "major"},
		{"C", "phrygian"},
		{"C", ""},
	}
	for _, tc := range cases {
		if got := ToCamelot(tc.key, tc.scale); got != "" {
			t.Errorf("ToCamelot(%q, %q) = %q, want empty", tc.key, tc.scale, got)
		}
	}
}
