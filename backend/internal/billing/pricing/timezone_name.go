// 时区名称必须非空，且不能使用依赖进程设置的 Local。
package pricing

import (
	"fmt"
	"strings"
)

func ValidateTimezoneName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("timezone is required")
	}
	if name == "Local" {
		return fmt.Errorf("local is not a supported timezone")
	}
	return nil
}
