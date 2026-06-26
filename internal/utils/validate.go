package utils

import (
	"errors"
	"net/url"
	"strings"
)

// maxLongURLLength 长链接最大允许长度
const maxLongURLLength = 10000

// maxShortKeyLength 自定义短链接key最大允许长度
const maxShortKeyLength = 32

// ValidateShortKey 校验自定义短链接key是否合法。
// 空字符串表示由系统自动生成，允许通过；非空时必须不超过32位，且字符集与自动生成key一致。
func ValidateShortKey(key string) error {
	if key == "" {
		return nil
	}
	if len(key) > maxShortKeyLength {
		return errors.New("short key exceeds max length")
	}
	for _, ch := range key {
		if !strings.ContainsRune(letterBytes, ch) {
			return errors.New("short key contains invalid character")
		}
	}
	return nil
}

// ValidateLongURL 校验待缩短的长链接是否合法。
// 仅允许 http/https scheme，拒绝 javascript:/data: 等危险scheme，避免短链服务被用作钓鱼/开放重定向跳板。
func ValidateLongURL(raw string) error {
	if raw == "" {
		return errors.New("long url is empty")
	}
	if len(raw) > maxLongURLLength {
		return errors.New("long url exceeds max length")
	}

	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return errors.New("long url is not a valid absolute URL")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("long url scheme must be http or https")
	}

	if parsed.Host == "" {
		return errors.New("long url host is empty")
	}

	return nil
}
