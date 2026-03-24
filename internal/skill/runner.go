package skill

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Runner 在隔离进程中执行 Skill
type Runner struct {
	skillDir string
	logger   *slog.Logger
	timeout  time.Duration
}

// NewRunner 创建 Skill 执行器
func NewRunner(skillDir string, logger *slog.Logger) *Runner {
	if err := os.MkdirAll(skillDir, 0700); err != nil {
		logger.Error("创建 Skill 目录失败", "error", err)
	}
	return &Runner{
		skillDir: skillDir,
		logger:   logger,
		timeout:  30 * time.Second,
	}
}

// RunResult Skill 执行结果
type RunResult struct {
	Output   string        `json:"output"`
	Error    string        `json:"error,omitempty"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
}

// Run 在隔离进程中执行 Skill
func (r *Runner) Run(ctx context.Context, m Manifest, input string, env map[string]string) (*RunResult, error) {
	start := time.Now()

	entrypoint := filepath.Join(r.skillDir, m.Name, m.Entrypoint)
	if _, err := os.Stat(entrypoint); err != nil {
		return nil, fmt.Errorf("Skill 入口不存在: %s", entrypoint)
	}

	// 构建命令
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var cmd *exec.Cmd

	if runtime.GOOS == "darwin" {
		// macOS: 用 sandbox-exec 限制（基础沙箱）
		profile := r.buildSandboxProfile(m)
		cmd = exec.CommandContext(ctx, "sandbox-exec", "-p", profile, entrypoint)
	} else {
		// Linux: 直接执行（P1 加 seccomp/namespace）
		cmd = exec.CommandContext(ctx, entrypoint)
	}

	// 设置环境变量——只传入声明的
	cmd.Env = r.buildEnv(m, env)

	// 通过 stdin 传入输入
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 工作目录隔离
	workDir := filepath.Join(r.skillDir, m.Name)
	cmd.Dir = workDir

	r.logger.Info("执行 Skill",
		"name", m.Name,
		"entrypoint", m.Entrypoint,
		"timeout", r.timeout,
	)

	err := cmd.Run()
	duration := time.Since(start)

	result := &RunResult{
		Output:   stdout.String(),
		Duration: duration,
	}

	if err != nil {
		result.Error = stderr.String()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		r.logger.Warn("Skill 执行失败",
			"name", m.Name,
			"exit_code", result.ExitCode,
			"stderr", result.Error,
			"duration", duration,
		)
	} else {
		r.logger.Info("Skill 执行完成",
			"name", m.Name,
			"output_len", len(result.Output),
			"duration", duration,
		)
	}

	return result, nil
}

// buildEnv 构建受限的环境变量
func (r *Runner) buildEnv(m Manifest, extra map[string]string) []string {
	env := []string{
		"PATH=/usr/bin:/bin:/usr/local/bin",
		"HOME=" + filepath.Join(r.skillDir, m.Name),
		"LANG=en_US.UTF-8",
	}

	// 只传入 Skill 声明需要的凭证
	for _, cred := range m.Permissions.Credentials {
		if val, ok := extra[cred]; ok {
			env = append(env, cred+"="+val)
		}
	}

	return env
}

// buildSandboxProfile macOS sandbox-exec 配置
func (r *Runner) buildSandboxProfile(m Manifest) string {
	var rules []string

	// 基础：允许进程执行
	rules = append(rules, "(allow process-exec)")
	rules = append(rules, "(allow process-fork)")

	// 文件系统
	switch m.Permissions.Filesystem {
	case "read":
		rules = append(rules, fmt.Sprintf(`(allow file-read* (subpath "%s"))`, filepath.Join(r.skillDir, m.Name)))
		rules = append(rules, "(deny file-write*)")
	case "write":
		rules = append(rules, fmt.Sprintf(`(allow file-read* (subpath "%s"))`, filepath.Join(r.skillDir, m.Name)))
		rules = append(rules, fmt.Sprintf(`(allow file-write* (subpath "%s"))`, filepath.Join(r.skillDir, m.Name)))
	default: // "none"
		rules = append(rules, "(deny file-read* file-write*)")
		// 但要允许读取必要的系统库
		rules = append(rules, `(allow file-read* (subpath "/usr/lib"))`)
		rules = append(rules, `(allow file-read* (subpath "/usr/share"))`)
		rules = append(rules, `(allow file-read* (subpath "/System"))`)
	}

	// 网络
	if len(m.Permissions.Network) > 0 {
		rules = append(rules, "(allow network-outbound)")
	} else {
		rules = append(rules, "(deny network*)")
	}

	profile := "(version 1)\n(deny default)\n" + strings.Join(rules, "\n")
	return profile
}

// ListInstalled 列出已安装的 Skill
func (r *Runner) ListInstalled() ([]Manifest, error) {
	entries, err := os.ReadDir(r.skillDir)
	if err != nil {
		return nil, err
	}

	var manifests []Manifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		mPath := filepath.Join(r.skillDir, e.Name(), "skill.yaml")
		data, err := os.ReadFile(mPath)
		if err != nil {
			continue
		}
		var m Manifest
		if err := parseManifest(data, &m); err != nil {
			continue
		}
		manifests = append(manifests, m)
	}

	return manifests, nil
}
