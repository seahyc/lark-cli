package cmd

import "testing"

func TestClassifyMemberIdent(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"ou_abcd1234", "open_id"},
		{"alice@example.com", "email"},
		{"Alice", "name"},
		{"Alice Wong", "name"},
		// ou_ prefix wins over @.
		{"ou_abc@ignored.com", "open_id"},
		// Empty input falls through as a "name" (caller handles it).
		{"", "name"},
	}
	for _, tc := range cases {
		if got := classifyMemberIdent(tc.in); got != tc.want {
			t.Errorf("classifyMemberIdent(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
