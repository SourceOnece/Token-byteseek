package team

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/TokenFlux/TokenRouter/internal/settings"
)

// AdminSettings 只包含本模块的管理开关与运行参数。
type AdminSettings struct {
	TeamEnabled                   bool `json:"team_enabled"`
	TeamInvitationCooldownSeconds *int `json:"team_invitation_cooldown_seconds,omitempty"`
	TeamInvitationHourlyLimit     *int `json:"team_invitation_hourly_limit,omitempty"`
}

// 本模块继续使用现有设置键。
const (
	SettingKeyTeamEnabled = "team_enabled"
)

// PrepareAdminSettings 保留既有编码和合法范围，不启动任务或发布成功通知。
func PrepareAdminSettings(value AdminSettings) (AdminSettings, map[string]string, error) {
	values := map[string]string{}
	values[SettingKeyTeamEnabled] = strconv.FormatBool(value.TeamEnabled)
	limits := DefaultTeamInvitationRateLimits()
	if value.TeamInvitationCooldownSeconds != nil {
		limits.CooldownSeconds = *value.TeamInvitationCooldownSeconds
		values[SettingKeyTeamInvitationCooldownSeconds] = strconv.Itoa(limits.CooldownSeconds)
	}
	if value.TeamInvitationHourlyLimit != nil {
		limits.HourlyLimit = *value.TeamInvitationHourlyLimit
		values[SettingKeyTeamInvitationHourlyLimit] = strconv.Itoa(limits.HourlyLimit)
	}
	if err := limits.Validate(); err != nil {
		return value, nil, err
	}

	return value, values, nil
}

// SettingsParticipant 静态声明写入所有权，只准备已经投影的实际字段。
func SettingsParticipant() settings.Participant {
	fields := []string{"team_enabled", "team_invitation_cooldown_seconds", "team_invitation_hourly_limit"}
	keys := []string{SettingKeyTeamEnabled, SettingKeyTeamInvitationCooldownSeconds, SettingKeyTeamInvitationHourlyLimit}
	return settings.Participant{Module: "team", Fields: fields, Keys: keys, Prepare: func(_ context.Context, input settings.Fields, _ map[string]string) (settings.PreparedChange, error) {
		if len(input) == 0 {
			return settings.PreparedChange{}, nil
		}
		raw, err := json.Marshal(input)
		if err != nil {
			return settings.PreparedChange{}, err
		}
		var value AdminSettings
		if err = json.Unmarshal(raw, &value); err != nil {
			return settings.PreparedChange{}, err
		}
		_, values, err := PrepareAdminSettings(value)
		if err != nil {
			return settings.PreparedChange{}, err
		}
		for i, field := range fields {
			if _, ok := input[field]; !ok {
				delete(values, keys[i])
			}
		}
		return settings.PreparedChange{Values: values}, nil
	}}
}
