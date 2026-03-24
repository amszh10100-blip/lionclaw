//go:build darwin

package vault

import (
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
)

// keychainGet 从 macOS Keychain 读取（hex 编码）
func keychainGet(service, account string) ([]byte, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", service,
		"-a", account,
		"-w",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Keychain 读取失败: %w", err)
	}
	hexStr := strings.TrimSpace(string(out))
	return hex.DecodeString(hexStr)
}

// keychainSet 存储到 macOS Keychain（hex 编码）
func keychainSet(service, account string, value []byte) error {
	hexStr := hex.EncodeToString(value)

	// 先删旧值
	exec.Command("security", "delete-generic-password",
		"-s", service, "-a", account,
	).Run()

	cmd := exec.Command("security", "add-generic-password",
		"-s", service,
		"-a", account,
		"-w", hexStr,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Keychain 写入失败: %w", err)
	}
	return nil
}
