// Package crypto 提供 API Key 的加密存取能力。
// 设计原则：
//   - API Key 只以密文形式存入数据库
//   - 解密后的明文只存在于 goroutine 栈上，不写入任何持久化介质
//   - 加密密钥从环境变量读取，不入代码库
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

var (
	ErrKeyNotSet   = errors.New("ENCRYPTION_KEY environment variable not set")
	ErrKeyInvalid  = errors.New("ENCRYPTION_KEY must be 32-byte hex string (64 hex chars)")
	ErrDecryptFail = errors.New("decryption failed: invalid ciphertext or wrong key")
)

// Keystore 管理 API Key 的加解密
type Keystore struct {
	key []byte // 32字节 AES-256 密钥，从环境变量加载
}

var (
	globalKeystore *Keystore
	once           sync.Once
	initErr        error
)

// Global 返回全局 Keystore 单例（线程安全）
func Global() (*Keystore, error) {
	once.Do(func() {
		globalKeystore, initErr = newFromEnv()
	})
	return globalKeystore, initErr
}

// newFromEnv 从环境变量 ENCRYPTION_KEY 创建 Keystore
func newFromEnv() (*Keystore, error) {
	hexKey := os.Getenv("ENCRYPTION_KEY")
	if hexKey == "" {
		return nil, ErrKeyNotSet
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != 32 {
		return nil, ErrKeyInvalid
	}
	return &Keystore{key: key}, nil
}

// NewWithKey 用指定密钥创建 Keystore（测试专用）
func NewWithKey(hexKey string) (*Keystore, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != 32 {
		return nil, ErrKeyInvalid
	}
	return &Keystore{key: key}, nil
}

// Encrypt 加密 API Key，返回 hex 编码的密文
// 格式：hex(nonce + ciphertext + tag)
// nonce 每次随机生成（GCM 标准），同一明文每次密文不同
func (k *Keystore) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(k.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	// 随机 nonce（12字节，GCM 标准）
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	// 加密：nonce 拼接在密文头部，方便解密时提取
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt 解密 API Key，返回明文
// ⚠️ 调用方注意：返回的明文不得写入日志或持久化存储
func (k *Keystore) Decrypt(ciphertextHex string) (string, error) {
	data, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", ErrDecryptFail
	}

	block, err := aes.NewCipher(k.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrDecryptFail
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrDecryptFail
	}

	return string(plaintext), nil
}

// Hint 返回 API Key 的掩码显示（用于前端展示，不泄露完整 Key）
// 示例：sk-ant-api03-xxxxx...yyyy → sk-ant-a...yyyy
func Hint(apiKey string) string {
	if len(apiKey) < 12 {
		return "***"
	}
	prefix := apiKey[:8]
	suffix := apiKey[len(apiKey)-4:]
	return prefix + "..." + suffix
}
