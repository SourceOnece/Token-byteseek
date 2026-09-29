package postgres

import (
	"testing"
	"time"

	dbent "github.com/TokenFlux/TokenRouter/ent"
	"github.com/stretchr/testify/require"
)

func TestFromAccountEntityPreservesProviderRecordAndProxy(t *testing.T) {
	notes := "note"
	username := "proxy-user"
	password := "proxy-password"
	parent := int64(11)
	rate := 0.0
	created := time.Date(2026, 9, 29, 1, 2, 3, 0, time.UTC)
	entity := &dbent.Account{
		ID: 7, Name: "alpha", Notes: &notes, Platform: "openai", Type: "oauth",
		Credentials: map[string]any{"access_token": "secret"}, Extra: map[string]any{"ticket": true},
		Concurrency: 3, Priority: 20, RateMultiplier: rate, Status: "active", Schedulable: true,
		CreatedAt: created, UpdatedAt: created, ParentAccountID: &parent,
		Edges: dbent.AccountEdges{Proxy: &dbent.Proxy{ID: 3, Name: "P", Protocol: "http", Host: "proxy.example", Port: 8080, Username: &username, Password: &password, Status: "active", CreatedAt: created, UpdatedAt: created}},
	}
	record := FromAccountEntity(entity)
	require.Equal(t, int64(7), record.ID)
	require.Equal(t, "alpha", record.Name)
	require.Equal(t, &parent, record.ParentProviderID)
	require.Equal(t, &rate, record.RateMultiplier)
	require.Equal(t, "secret", record.Credentials["access_token"])
	require.Equal(t, int64(3), record.Proxy.ID)
	require.Equal(t, username, record.Proxy.Username)
	require.Equal(t, password, record.Proxy.Password)
	// 记录类型不能通过 fmt/String 暴露凭据或代理。
	require.NotContains(t, record.String(), "secret")
}
