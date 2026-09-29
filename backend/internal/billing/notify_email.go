package billing

import (
	"github.com/TokenFlux/TokenRouter/internal/identity/contact"
)

// ParseNotifyEmails 委托身份邮箱格式兼容。
func ParseNotifyEmails(raw string) []NotifyEmailSummary { return contact.ParseNotifyEmails(raw) }
