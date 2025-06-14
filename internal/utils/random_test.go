package utils

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{
			name:   "zero length",
			length: 0,
		},
		{
			name:   "single character",
			length: 1,
		},
		{
			name:   "short string",
			length: 5,
		},
		{
			name:   "default handler length",
			length: 7,
		},
		{
			name:   "medium length",
			length: 16,
		},
		{
			name:   "long string",
			length: 64,
		},
		{
			name:   "very long string",
			length: 128,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateRandomString(tt.length)

			// 验证长度
			assert.Equal(t, tt.length, len(result), "Generated string length should match requested length")

			// 验证字符集 - 只包含允许的字符
			validChars := regexp.MustCompile(`^[0-9a-zA-Z]*$`)
			assert.True(t, validChars.MatchString(result), "Generated string should only contain alphanumeric characters")
		})
	}
}

func TestGenerateRandomStringCharacterSet(t *testing.T) {
	// 生成足够长的字符串以确保覆盖大部分字符
	length := 1000
	result := GenerateRandomString(length)

	// 验证字符集
	expectedChars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	for _, char := range result {
		assert.Contains(t, expectedChars, string(char), "Generated character should be in the expected character set")
	}

	// 验证包含不同类型的字符（数字、小写字母、大写字母）
	// 注意：由于随机性，这个测试有极小概率失败，但在1000个字符中几乎不可能
	hasDigit := false
	hasLower := false
	hasUpper := false

	for _, char := range result {
		switch {
		case char >= '0' && char <= '9':
			hasDigit = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		}
	}

	// 在1000个字符中，应该包含所有类型的字符
	assert.True(t, hasDigit, "Generated string should contain digits")
	assert.True(t, hasLower, "Generated string should contain lowercase letters")
	assert.True(t, hasUpper, "Generated string should contain uppercase letters")
}

func TestGenerateRandomStringUniqueness(t *testing.T) {
	// 测试生成的字符串的唯一性
	length := 10
	iterations := 1000
	generated := make(map[string]bool)
	duplicates := 0

	for i := 0; i < iterations; i++ {
		result := GenerateRandomString(length)
		if generated[result] {
			duplicates++
		}
		generated[result] = true
	}

	// 在10位长度的字符串中，1000次生成应该有很高的唯一性
	// 允许少量重复（由于真正的随机性），但不应该超过总数的5%
	duplicateRate := float64(duplicates) / float64(iterations)
	assert.Less(t, duplicateRate, 0.05, "Duplicate rate should be less than 5%")

	// 至少应该生成超过90%的唯一字符串
	uniqueCount := len(generated)
	uniqueRate := float64(uniqueCount) / float64(iterations)
	assert.Greater(t, uniqueRate, 0.90, "Should generate mostly unique strings")
}

func TestGenerateRandomStringDistribution(t *testing.T) {
	// 测试字符分布的相对均匀性
	length := 5000 // 足够大的样本
	result := GenerateRandomString(length)

	// 统计每个字符的出现次数
	charCount := make(map[rune]int)
	expectedChars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	// 初始化计数器
	for _, char := range expectedChars {
		charCount[char] = 0
	}

	// 统计实际字符分布
	for _, char := range result {
		charCount[char]++
	}

	// 期望的平均频率
	expectedAverage := float64(length) / float64(len(expectedChars))

	// 检查每个字符的频率是否在合理范围内
	// 允许±50%的偏差（由于随机性）
	tolerance := expectedAverage * 0.5

	for char, count := range charCount {
		frequency := float64(count)
		assert.True(t,
			frequency >= expectedAverage-tolerance && frequency <= expectedAverage+tolerance,
			"Character '%c' frequency (%f) should be within reasonable range of expected average (%f)",
			char, frequency, expectedAverage)
	}
}

func TestGenerateRandomStringNegativeLength(t *testing.T) {
	// 测试负数长度（边界情况）
	// 当前实现会panic，这是Go make()函数的预期行为
	assert.Panics(t, func() {
		GenerateRandomString(-1)
	}, "Negative length should cause panic")

	assert.Panics(t, func() {
		GenerateRandomString(-10)
	}, "Negative length should cause panic")
}

func TestGenerateRandomStringConsistentCharacterSet(t *testing.T) {
	// 验证使用的字符集与常量letterBytes一致
	length := 1000
	result := GenerateRandomString(length)

	for _, char := range result {
		assert.Contains(t, letterBytes, string(char),
			"Generated character should be from the letterBytes constant")
	}
}

func TestLetterBytesConstant(t *testing.T) {
	// 测试letterBytes常量的正确性
	expected := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	assert.Equal(t, expected, letterBytes, "letterBytes constant should contain correct character set")

	// 验证字符集的完整性
	assert.Equal(t, 62, len(letterBytes), "letterBytes should contain 62 characters (10 digits + 26 lowercase + 26 uppercase)")

	// 验证没有重复字符
	charSet := make(map[rune]bool)
	for _, char := range letterBytes {
		assert.False(t, charSet[char], "letterBytes should not contain duplicate characters")
		charSet[char] = true
	}
}

// 基准测试
func BenchmarkGenerateRandomString(b *testing.B) {
	lengths := []int{1, 5, 10, 50, 100}

	for _, length := range lengths {
		b.Run(fmt.Sprintf("length_%d", length), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				GenerateRandomString(length)
			}
		})
	}
}

func BenchmarkGenerateRandomStringParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			GenerateRandomString(10)
		}
	})
}
