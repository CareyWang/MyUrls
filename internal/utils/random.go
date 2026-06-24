package utils

import (
	"crypto/rand"
	"math/big"
)

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// GenerateRandomString 生成长度为 bits 的随机字符串，字符取自 letterBytes。
// 使用 crypto/rand 而非 math/rand：math/rand 按纳秒时间播种，高并发下可能在同一纳秒内
// 重复播种导致生成相同的序列（短链key可预测/碰撞），crypto/rand 不存在该问题。
func GenerateRandomString(bits int) string {
	b := make([]byte, bits)
	max := big.NewInt(int64(len(letterBytes)))

	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		b[i] = letterBytes[n.Int64()]
	}

	return string(b)
}
