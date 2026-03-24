//go:build linux

package vault

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/user"
)

// Linux: 使用环境变量或机器特征派生主密钥
// 不再将主密钥明文落盘

func keychainGet(service, account string) ([]byte, error) {
	if key := os.Getenv("LIONCLAW_MASTER_KEY"); key != "" {
		hash := sha256.Sum256([]byte(key))
		return hash[:], nil
	}

	hostname, _ := os.Hostname()
	u, _ := user.Current()
	uid := u.Uid

	// 基于机器特征派生，不落盘（存在变更风险，但符合最低安全要求）
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%s-%s-lionclaw-salt", hostname, uid, service, account)))
	return hash[:], nil
}

func keychainSet(service, account string, value []byte) error {
	// 不再将主密钥明文写入磁盘
	return nil
}
