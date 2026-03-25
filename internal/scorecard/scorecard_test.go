package scorecard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerate(t *testing.T) {
	tmpDir := t.TempDir()

	cfgFile := filepath.Join(tmpDir, "openclaw.json")
	if err := os.WriteFile(cfgFile, []byte(`{"apiKey": "test", "bind": "0.0.0.0"}`), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	card := Generate(tmpDir)
	if card == nil {
		t.Fatalf("Generate() returned nil")
	}

	if len(card.Items) == 0 {
		t.Errorf("Generate() returned empty items")
	}

	format := card.Format()
	if format == "" {
		t.Errorf("Format() returned empty string")
	}

	cardNoDir := Generate("")
	if cardNoDir == nil {
		t.Fatalf("Generate(\"\") returned nil")
	}
}

func TestScoreToGrade(t *testing.T) {
	tests := []struct {
		score    int
		expected string
	}{
		{95, "A+"},
		{90, "A"},
		{85, "B+"},
		{75, "B"},
		{65, "C"},
		{50, "D"},
	}

	for _, tt := range tests {
		grade := scoreToGrade(tt.score)
		if grade != tt.expected {
			t.Errorf("scoreToGrade(%d) = %s; want %s", tt.score, grade, tt.expected)
		}
	}
}
