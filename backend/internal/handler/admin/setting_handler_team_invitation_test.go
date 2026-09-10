//go:build unit

package admin

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUpdateSettingsTeamInvitationLimits(t *testing.T) {
	h, repo := newStepUpSwitchTestHandler(t, map[string]string{service.SettingKeyTeamInvitationCooldownSeconds: "60", service.SettingKeyTeamInvitationHourlyLimit: "20"})
	rec := doUpdateSettings(t, h, map[string]any{"team_invitation_cooldown_seconds": 10, "team_invitation_hourly_limit": 100}, nil)
	require.Equal(t, 200, rec.Code, rec.Body.String())
	require.Equal(t, "10", repo.values[service.SettingKeyTeamInvitationCooldownSeconds])
	require.Equal(t, "100", repo.values[service.SettingKeyTeamInvitationHourlyLimit])
	var body struct {
		Data struct {
			Cooldown int `json:"team_invitation_cooldown_seconds"`
			Hourly   int `json:"team_invitation_hourly_limit"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 10, body.Data.Cooldown)
	require.Equal(t, 100, body.Data.Hourly)
	getRec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(getRec)
	c.Request = httptest.NewRequest("GET", "/api/v1/admin/settings", nil)
	h.GetSettings(c)
	require.Equal(t, 200, getRec.Code)
	require.NoError(t, json.Unmarshal(getRec.Body.Bytes(), &body))
	require.Equal(t, 10, body.Data.Cooldown)
	require.Equal(t, 100, body.Data.Hourly)
	rec = doUpdateSettings(t, h, map[string]any{"team_enabled": false}, nil)
	require.Equal(t, 200, rec.Code)
	require.Equal(t, "10", repo.values[service.SettingKeyTeamInvitationCooldownSeconds])
	require.Equal(t, "100", repo.values[service.SettingKeyTeamInvitationHourlyLimit])
	for _, key := range []string{"team_invitation_cooldown_seconds", "team_invitation_hourly_limit"} {
		for _, value := range []any{0, -1, 100000, 1.5, "oops"} {
			rec = doUpdateSettings(t, h, map[string]any{key: value}, nil)
			require.Equal(t, 400, rec.Code)
			require.Equal(t, "10", repo.values[service.SettingKeyTeamInvitationCooldownSeconds])
			require.Equal(t, "100", repo.values[service.SettingKeyTeamInvitationHourlyLimit])
		}
	}
}

func TestUpdateSettingsTeamInvitationAudit(t *testing.T) {
	before := &service.SystemSettings{TeamInvitationCooldownSeconds: 60, TeamInvitationHourlyLimit: 20}
	after := &service.SystemSettings{TeamInvitationCooldownSeconds: 10, TeamInvitationHourlyLimit: 100}
	changes := diffSettings(before, after, nil, nil, UpdateSettingsRequest{})
	require.Contains(t, changes, "team_invitation_cooldown_seconds")
	require.Contains(t, changes, "team_invitation_hourly_limit")
}
