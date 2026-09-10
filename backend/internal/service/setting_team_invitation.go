package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

const (
	DefaultTeamInvitationCooldownSeconds = 60
	DefaultTeamInvitationHourlyLimit     = 20
	MaxTeamInvitationCooldownSeconds     = 86400
	MaxTeamInvitationHourlyLimit         = 10000
)

// TeamInvitationRateLimits 只用于平台团队邀请，不改变其他邮件场景。
type TeamInvitationRateLimits struct {
	CooldownSeconds int
	HourlyLimit     int
}

func DefaultTeamInvitationRateLimits() TeamInvitationRateLimits {
	return TeamInvitationRateLimits{DefaultTeamInvitationCooldownSeconds, DefaultTeamInvitationHourlyLimit}
}

func (v TeamInvitationRateLimits) Validate() error {
	if v.CooldownSeconds < 1 || v.CooldownSeconds > MaxTeamInvitationCooldownSeconds {
		return fmt.Errorf("同邮箱邀请间隔须为 1–%d 秒", MaxTeamInvitationCooldownSeconds)
	}
	if v.HourlyLimit < 1 || v.HourlyLimit > MaxTeamInvitationHourlyLimit {
		return fmt.Errorf("每团队每小时邀请上限须为 1–%d 次", MaxTeamInvitationHourlyLimit)
	}
	return nil
}

// 历史缺键或损坏值回退升级前默认，不能将 0 静默解释成无限制。
func parseTeamInvitationRateLimits(values map[string]string) TeamInvitationRateLimits {
	limits := DefaultTeamInvitationRateLimits()
	if n, err := strconv.Atoi(strings.TrimSpace(values[SettingKeyTeamInvitationCooldownSeconds])); err == nil && n >= 1 && n <= MaxTeamInvitationCooldownSeconds {
		limits.CooldownSeconds = n
	}
	if n, err := strconv.Atoi(strings.TrimSpace(values[SettingKeyTeamInvitationHourlyLimit])); err == nil && n >= 1 && n <= MaxTeamInvitationHourlyLimit {
		limits.HourlyLimit = n
	}
	return limits
}

// GetTeamInvitationRateLimits 一次读取两键，无进程缓存，多实例下次请求即可使用新配置。
// @project-doc docs/interfaces/configuration.md#team_invitation_limits
func (s *SettingService) GetTeamInvitationRateLimits(ctx context.Context) (TeamInvitationRateLimits, error) {
	if s == nil {
		return DefaultTeamInvitationRateLimits(), nil
	}
	values, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeyTeamInvitationCooldownSeconds, SettingKeyTeamInvitationHourlyLimit})
	if err != nil {
		return TeamInvitationRateLimits{}, err
	}
	return parseTeamInvitationRateLimits(values), nil
}
