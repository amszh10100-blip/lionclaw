package updater

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"v1.0.0", "v1.0.0", 0},
		{"v1.0.0", "v1.0.1", -1},
		{"v1.0.1", "v1.0.0", 1},
		{"v2.0.0", "v1.9.9", 1},
		{"v1.10.0", "v1.2.0", 1},
		{"1.0.0", "v1.0.0", 0},
	}

	for _, tt := range tests {
		result := CompareVersions(tt.v1, tt.v2)
		if result != tt.expected {
			t.Errorf("CompareVersions(%q, %q) = %d; want %d", tt.v1, tt.v2, result, tt.expected)
		}
	}
}

func TestUpdater_Update(t *testing.T) {
	tmpDir := t.TempDir()

	newBinary := filepath.Join(tmpDir, "new_lionclaw")
	if err := os.WriteFile(newBinary, []byte("dummy binary"), 0755); err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	// Just a quick check on missing files
	u := &Updater{installDir: tmpDir}
	err := u.Update(context.Background(), "nonexistent_file")
	if err == nil {
		t.Errorf("Update() expected error for nonexistent file")
	}
}

func TestCopyFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.txt")
	dst := filepath.Join(t.TempDir(), "dst.txt")

	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatalf("Failed to write src: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Errorf("copyFile() failed: %v", err)
	}

	// Error case
	if err := copyFile("nonexistent", dst); err == nil {
		t.Errorf("copyFile() expected error for nonexistent file")
	}
}
