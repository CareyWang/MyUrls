package utils

import (
	"errors"
	"net/url"
)

// maxLongURLLength 长链接最大允许长度
const maxLongURLLength = 10000

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
