package auth

import "testing"

func TestExtractCode(t *testing.T) {
	const state = "abc123"

	tests := []struct {
		name    string
		pasted  string
		want    string
		wantErr bool
	}{
		{"bare code", "XYZ-code-value", "XYZ-code-value", false},
		{"full url with matching state",
			"http://localhost:53219/callback?code=THE_CODE&state=abc123", "THE_CODE", false},
		{"full url without state (still ok)",
			"http://localhost:53219/callback?code=THE_CODE", "THE_CODE", false},
		{"state mismatch is rejected",
			"http://localhost:53219/callback?code=THE_CODE&state=evil", "", true},
		{"url missing code", "http://localhost:53219/callback?state=abc123", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractCode(tt.pasted, state)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got code=%q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
