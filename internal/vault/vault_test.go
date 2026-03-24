package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVaultSetGetDelete(t *testing.T) {
	// 使用临时目录
	dir := t.TempDir()

	v, err := NewFileVault(dir)
	if err != nil {
		t.Fatalf("NewFileVault failed: %v", err)
	}

	// Set
	if err := v.Set("TEST_KEY", []byte("secret_value")); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Has
	if !v.Has("TEST_KEY") {
		t.Error("Has(TEST_KEY) = false, want true")
	}
	if v.Has("NONEXIST") {
		t.Error("Has(NONEXIST) = true, want false")
	}

	// Get
	val, err := v.Get("TEST_KEY")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(val) != "secret_value" {
		t.Errorf("Get = %q, want %q", string(val), "secret_value")
	}

	// List
	keys, _ := v.List()
	if len(keys) != 1 || keys[0] != "TEST_KEY" {
		t.Errorf("List = %v, want [TEST_KEY]", keys)
	}

	// Delete
	if err := v.Delete("TEST_KEY"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if v.Has("TEST_KEY") {
		t.Error("After delete, Has(TEST_KEY) = true")
	}
}

func TestVaultEncryptionOnDisk(t *testing.T) {
	dir := t.TempDir()

	v, _ := NewFileVault(dir)
	v.Set("SECRET", []byte("super_secret_password_12345"))

	// 读取磁盘文件
	encFile := filepath.Join(dir, "vault.enc")
	data, err := os.ReadFile(encFile)
	if err != nil {
		t.Fatalf("读取加密文件失败: %v", err)
	}

	// 确认明文不在磁盘上
	if containsBytes(data, []byte("super_secret_password_12345")) {
		t.Error("明文密码出现在加密文件中！")
	}
	if containsBytes(data, []byte("SECRET")) {
		t.Error("明文 key 名出现在加密文件中！")
	}
}

func TestVaultMultipleKeys(t *testing.T) {
	dir := t.TempDir()
	v, _ := NewFileVault(dir)

	v.Set("KEY1", []byte("val1"))
	v.Set("KEY2", []byte("val2"))
	v.Set("KEY3", []byte("val3"))

	keys, _ := v.List()
	if len(keys) != 3 {
		t.Errorf("List len = %d, want 3", len(keys))
	}

	// 重新加载
	v2, _ := NewFileVault(dir)
	val, _ := v2.Get("KEY2")
	if string(val) != "val2" {
		t.Errorf("Reload Get(KEY2) = %q, want val2", string(val))
	}
}

func TestVaultGetNonExistent(t *testing.T) {
	dir := t.TempDir()
	v, _ := NewFileVault(dir)

	_, err := v.Get("NOPE")
	if err == nil {
		t.Error("Get(NOPE) should return error")
	}
}

func containsBytes(data, sub []byte) bool {
	for i := 0; i <= len(data)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			if data[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
