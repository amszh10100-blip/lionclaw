//go:build linux

package vault

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Linux: 使用文件存储主密钥（权限 0600）
// TODO: P1 升级为 libsecret (GNOME Keyring)

func keychainGet(service, account string) ([]byte, error) {
	path := keyfilePath(service, account)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取密钥文件失败: %w", err)
	}
	return data, nil
}

func keychainSet(service, account string, value []byte) error {
	path := keyfilePath(service, account)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(path, value, 0600)
}

func keyfilePath(service, account string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".goldlion", ".keystore", service+"-"+account)
}

// 确保有随机源
func init() {
	_ = rand.Reader
	_ = io.ReadFull
}
