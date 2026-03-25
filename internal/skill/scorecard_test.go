package skill

import "testing"

func TestSecurityScore(t *testing.T) {
	tests := []struct {
		warnings int
		errors   int
		expected string
	}{
		{0, 0, "A"},
		{1, 0, "B"},
		{2, 0, "B"},
		{3, 0, "C"},
		{0, 1, "D"},
		{1, 1, "D"},
		{0, 2, "F"},
		{0, 5, "F"},
	}

	for _, tt := range tests {
		score := SecurityScore(tt.warnings, tt.errors)
		if score != tt.expected {
			t.Errorf("SecurityScore(%d, %d): expected %s, got %s", tt.warnings, tt.errors, tt.expected, score)
		}
	}
}
