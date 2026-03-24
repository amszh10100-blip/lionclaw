package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Vault 加密凭证存储接口
type Vault interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	List() ([]string, error)
	Has(key string) bool
}

// FileVault 基于文件的加密 Vault
// 使用 AES-256-GCM 加密，主密钥从 OS Keychain 获取
type FileVault struct {
	path string
	key  []byte // AES-256 主密钥（32 bytes）
	data map[string][]byte
	mu   sync.RWMutex
}

// NewFileVault 创建或打开 Vault
func NewFileVault(dir string) (*FileVault, error) {
	path := filepath.Join(dir, "vault.enc")

	// 获取或生成主密钥
	masterKey, err := getOrCreateMasterKey()
	if err != nil {
		return nil, fmt.Errorf("获取主密钥失败: %w", err)
	}

	v := &FileVault{
		path: path,
		key:  masterKey,
		data: make(map[string][]byte),
	}

	// 尝试加载已有数据
	if _, err := os.Stat(path); err == nil {
		if err := v.load(); err != nil {
			return nil, fmt.Errorf("加载 Vault 失败: %w", err)
		}
	}

	return v, nil
}

func (v *FileVault) Set(key string, value []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.data[key] = append([]byte(nil), value...) // 深拷贝
	return v.save()
}

func (v *FileVault) Get(key string) ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	val, ok := v.data[key]
	if !ok {
		return nil, fmt.Errorf("凭证不存在: %s", key)
	}
	return append([]byte(nil), val...), nil // 深拷贝
}

func (v *FileVault) Delete(key string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.data, key)
	return v.save()
}

func (v *FileVault) List() ([]string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	keys := make([]string, 0, len(v.data))
	for k := range v.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func (v *FileVault) Has(key string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	_, ok := v.data[key]
	return ok
}

// encrypt AES-256-GCM 加密
func (v *FileVault) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decrypt AES-256-GCM 解密
func (v *FileVault) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(v.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("密文太短")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (v *FileVault) save() error {
	// 序列化
	plaintext, err := json.Marshal(v.data)
	if err != nil {
		return err
	}

	// 加密
	encrypted, err := v.encrypt(plaintext)
	if err != nil {
		return err
	}

	// 写入文件（0600 权限）
	dir := filepath.Dir(v.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(v.path, encrypted, 0600)
}

func (v *FileVault) load() error {
	encrypted, err := os.ReadFile(v.path)
	if err != nil {
		return err
	}

	plaintext, err := v.decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("解密失败（主密钥可能不匹配）: %w", err)
	}

	return json.Unmarshal(plaintext, &v.data)
}

// getOrCreateMasterKey 从 OS Keychain 获取或生成主密钥
func getOrCreateMasterKey() ([]byte, error) {
	// 尝试从 Keychain 读取
	key, err := keychainGet("lionclaw", "master-key")
	if err == nil && len(key) == 32 {
		return key, nil
	}

	// 生成新密钥
	newKey := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return nil, fmt.Errorf("生成密钥失败: %w", err)
	}

	// 存入 Keychain
	if err := keychainSet("lionclaw", "master-key", newKey); err != nil {
		return nil, fmt.Errorf("存储密钥到 Keychain 失败: %w", err)
	}

	return newKey, nil
}
