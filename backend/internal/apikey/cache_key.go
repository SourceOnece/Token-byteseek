// 认证缓存键使用 API Key 摘要，避免将原始凭据写入缓存键名。
package apikey

import (
	"crypto/sha256"
	"encoding/hex"
)

func AuthCacheKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}
