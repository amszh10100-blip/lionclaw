package skill

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestRunner(t *testing.T) (*Runner, string) {
	tmpDir, err := os.MkdirTemp("", "skill_runner_test")
	if err != nil {
		t.Fatalf("MkdirTemp failed: %v", err)
	}
	
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(tmpDir, logger)
	return runner, tmpDir
}

func TestRunner_Timeout(t *testing.T) {
	runner, tmpDir := setupTestRunner(t)
	defer os.RemoveAll(tmpDir)

	skillName := "test_timeout"
	skillPath := filepath.Join(tmpDir, skillName)
	err := os.MkdirAll(skillPath, 0755)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	entrypoint := filepath.Join(skillPath, "run.sh")
	script := "#!/bin/sh\nsleep 2\necho 'done'"
	err = os.WriteFile(entrypoint, []byte(script), 0755)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	m := Manifest{
		Name:       skillName,
		Entrypoint: "run.sh",
	}

	// override timeout for testing
	runner.timeout = 100 * time.Millisecond

	ctx := context.Background()
	result, err := runner.Run(ctx, m, "", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ExitCode == 0 {
		t.Errorf("Expected non-zero exit code due to timeout")
	}
}

func TestRunner_EnvIsolation(t *testing.T) {
	runner, tmpDir := setupTestRunner(t)
	defer os.RemoveAll(tmpDir)

	skillName := "test_env"
	skillPath := filepath.Join(tmpDir, skillName)
	err := os.MkdirAll(skillPath, 0755)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	entrypoint := filepath.Join(skillPath, "run.sh")
	script := "#!/bin/sh\nenv"
	err = os.WriteFile(entrypoint, []byte(script), 0755)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	m := Manifest{
		Name:       skillName,
		Entrypoint: "run.sh",
	}

	os.Setenv("SECRET_LEAK", "should_not_be_seen")
	defer os.Unsetenv("SECRET_LEAK")

	ctx := context.Background()
	result, err := runner.Run(ctx, m, "my_input", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ExitCode == 0 {
		if strings.Contains(result.Output, "SECRET_LEAK") {
			t.Errorf("Environment variable leaked into sandbox!")
		}
		if !strings.Contains(result.Output, "SKILL_INPUT=my_input") {
			t.Errorf("SKILL_INPUT not found in environment")
		}
	}
}

func TestRunner_OutputCapture(t *testing.T) {
	runner, tmpDir := setupTestRunner(t)
	defer os.RemoveAll(tmpDir)

	skillName := "test_output"
	skillPath := filepath.Join(tmpDir, skillName)
	err := os.MkdirAll(skillPath, 0755)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	entrypoint := filepath.Join(skillPath, "run.sh")
	script := "#!/bin/sh\necho 'hello world'\n>&2 echo 'error output'"
	err = os.WriteFile(entrypoint, []byte(script), 0755)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	m := Manifest{
		Name:       skillName,
		Entrypoint: "run.sh",
	}

	ctx := context.Background()
	result, err := runner.Run(ctx, m, "", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ExitCode == 0 {
		if !strings.Contains(result.Output, "hello world") {
			t.Errorf("Output capture failed. Got: %s", result.Output)
		}
		if !strings.Contains(result.Error, "error output") {
			t.Errorf("Stderr capture failed. Got: %s", result.Error)
		}
	}
}
