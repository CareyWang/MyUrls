package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateRandomToken 生成一个安全的随机token
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateCacheToken 生成一个32字节的随机token用于缓存清除
func GenerateCacheToken() (string, error) {
	return GenerateRandomToken(32)
}