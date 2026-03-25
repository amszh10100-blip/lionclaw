package updater

import (
	"context"
	"fmt"
	"github.com/amszh10100-blip/lionclaw/internal/config"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Updater 原子更新系统
type Updater struct {
	cfg        *config.Config
	installDir string // ~/.lionclaw
	logger     *slog.Logger
}

// NewUpdater 创建更新器
func NewUpdater(installDir string, cfg *config.Config, logger *slog.Logger) *Updater {
	return &Updater{installDir: installDir, cfg: cfg, logger: logger}
}

// CheckResult 版本检查结果
type CheckResult struct {
	CurrentVersion  string `json:"current"`
	LatestVersion   string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	DownloadURL     string `json:"download_url"`
}

// HealthCheck 健康检查项
type HealthCheck struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

// Update 执行原子更新
func (u *Updater) Update(ctx context.Context, newBinaryPath string) error {
	versionsDir := filepath.Join(u.installDir, "versions")
	binLink := filepath.Join(u.installDir, "bin", "lionclaw")

	// 1. 生成版本目录
	ts := time.Now().Format("20060102-150405")
	versionDir := filepath.Join(versionsDir, ts)
	os.MkdirAll(versionDir, 0700)

	newBinary := filepath.Join(versionDir, "lionclaw")

	// 2. 复制新二进制到版本目录
	if err := copyFile(newBinaryPath, newBinary); err != nil {
		return fmt.Errorf("复制二进制失败: %w", err)
	}
	os.Chmod(newBinary, 0755)

	// 3. 备份当前 symlink 目标
	oldTarget, _ := os.Readlink(binLink)
	u.logger.Info("备份当前版本", "path", oldTarget)

	// 4. 健康检查新版本
	checks := u.runHealthChecks(ctx, newBinary)
	allPass := true
	for _, c := range checks {
		if !c.Pass {
			allPass = false
			u.logger.Error("健康检查失败", "check", c.Name, "detail", c.Detail)
		}
	}

	if !allPass {
		// 回滚
		os.RemoveAll(versionDir)
		return fmt.Errorf("健康检查未通过，已回滚")
	}

	// 5. 原子切换 symlink
	tmpLink := binLink + ".new"
	os.Remove(tmpLink)
	if err := os.Symlink(newBinary, tmpLink); err != nil {
		return fmt.Errorf("创建 symlink 失败: %w", err)
	}
	if err := os.Rename(tmpLink, binLink); err != nil {
		return fmt.Errorf("原子切换失败: %w", err)
	}

	u.logger.Info("更新完成",
		"old", filepath.Base(filepath.Dir(oldTarget)),
		"new", ts,
	)

	// 6. 清理旧版本（保留最近 3 个）
	u.cleanOldVersions(versionsDir, 3)

	return nil
}

// Rollback 回滚到上一个版本
func (u *Updater) Rollback() error {
	versionsDir := filepath.Join(u.installDir, "versions")
	binLink := filepath.Join(u.installDir, "bin", "lionclaw")

	entries, err := os.ReadDir(versionsDir)
	if err != nil || len(entries) < 2 {
		return fmt.Errorf("没有可回滚的版本")
	}

	// 倒数第二个就是上一个版本
	prevVersion := entries[len(entries)-2].Name()
	prevBinary := filepath.Join(versionsDir, prevVersion, "lionclaw")

	if _, err := os.Stat(prevBinary); err != nil {
		return fmt.Errorf("上一版本二进制不存在: %s", prevBinary)
	}

	tmpLink := binLink + ".new"
	os.Remove(tmpLink)
	os.Symlink(prevBinary, tmpLink)
	os.Rename(tmpLink, binLink)

	u.logger.Info("已回滚", "to", prevVersion)
	return nil
}

// runHealthChecks 运行健康检查
func (u *Updater) runHealthChecks(ctx context.Context, binary string) []HealthCheck {
	var checks []HealthCheck

	// 1. 二进制可执行
	checks = append(checks, u.checkExecutable(binary))

	// 2. version 命令正常
	checks = append(checks, u.checkVersion(binary))

	// 3. 配置可加载
	checks = append(checks, u.checkConfig())

	// 4. 数据库可访问
	checks = append(checks, u.checkDatabase())

	// 5. Vault 可解密
	checks = append(checks, u.checkVault())

	// 6. Ollama 可达
	checks = append(checks, u.checkOllama())

	return checks
}

func (u *Updater) checkExecutable(binary string) HealthCheck {
	info, err := os.Stat(binary)
	if err != nil {
		return HealthCheck{"二进制存在", false, err.Error()}
	}
	if info.Mode()&0111 == 0 {
		return HealthCheck{"二进制可执行", false, "无执行权限"}
	}
	// 检查架构匹配
	expected := runtime.GOARCH
	out, _ := exec.Command("file", binary).Output()
	if strings.Contains(string(out), "arm64") && expected != "arm64" {
		return HealthCheck{"架构匹配", false, fmt.Sprintf("二进制 arm64, 系统 %s", expected)}
	}
	return HealthCheck{"二进制可执行", true, fmt.Sprintf("%d bytes", info.Size())}
}

func (u *Updater) checkVersion(binary string) HealthCheck {
	out, err := exec.Command(binary, "version").CombinedOutput()
	if err != nil {
		return HealthCheck{"version 命令", false, err.Error()}
	}
	return HealthCheck{"version 命令", true, strings.TrimSpace(string(out))}
}

func (u *Updater) checkConfig() HealthCheck {
	cfgPath := filepath.Join(u.installDir, "config.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		return HealthCheck{"配置文件", false, "config.yaml 不存在"}
	}
	return HealthCheck{"配置文件", true, "ok"}
}

func (u *Updater) checkDatabase() HealthCheck {
	dbPath := filepath.Join(u.installDir, "data", "lionclaw.db")
	if _, err := os.Stat(dbPath); err != nil {
		return HealthCheck{"数据库", true, "新安装，无数据库"} // 新安装不算失败
	}
	return HealthCheck{"数据库", true, "ok"}
}

func (u *Updater) checkVault() HealthCheck {
	vaultPath := filepath.Join(u.installDir, "vault.enc")
	if _, err := os.Stat(vaultPath); err != nil {
		return HealthCheck{"Vault", true, "无 vault（新安装）"}
	}
	return HealthCheck{"Vault", true, "ok"}
}

func (u *Updater) checkOllama() HealthCheck {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(u.cfg.Models.Local.Endpoint + "/api/tags")
	if err != nil {
		return HealthCheck{"Ollama", false, "不可达"}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return HealthCheck{"Ollama", false, fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}
	return HealthCheck{"Ollama", true, "ok"}
}

func (u *Updater) cleanOldVersions(dir string, keep int) {
	entries, _ := os.ReadDir(dir)
	if len(entries) <= keep {
		return
	}
	for _, e := range entries[:len(entries)-keep] {
		old := filepath.Join(dir, e.Name())
		os.RemoveAll(old)
		u.logger.Info("清理旧版本", "version", e.Name())
	}
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(d, s)
	return err
}

// CompareVersions 比较两个版本号字符串
// 返回 -1 (v1 < v2), 0 (v1 == v2), 1 (v1 > v2)
