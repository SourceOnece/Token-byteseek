package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/apikey"
	"github.com/TokenFlux/TokenRouter/internal/identity/httpapi/authctx"
	"github.com/TokenFlux/TokenRouter/internal/routing"
	"github.com/TokenFlux/TokenRouter/internal/routing/accessview"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type rotationHTTPRepository struct {
	apiKeyHandlerSecurityRepoStub
	calls int
	err   error
}

func (r *rotationHTTPRepository) RotateCredential(_ context.Context, key *apikey.APIKey, _ string) error {
	r.calls++
	if r.err != nil {
		return r.err
	}
	r.keys[key.ID].Key = key.Key
	return nil
}

// TestRotateCredentialRoute 通过正式路由验证归属边界和新凭据响应。
func TestRotateCredentialRoute(t *testing.T) {
	managed := "creative_studio"
	for _, test := range []struct {
		name   string
		id     string
		userID int64
		err    error
		status int
	}{
		{name: "success", id: "7", userID: 3, status: http.StatusOK},
		{name: "unauthenticated", id: "7", status: http.StatusUnauthorized},
		{name: "invalid_id", id: "bad", userID: 3, status: http.StatusBadRequest},
		{name: "negative_id", id: "-1", userID: 3, status: http.StatusBadRequest},
		{name: "missing", id: "404", userID: 3, status: http.StatusNotFound},
		{name: "other_owner", id: "7", userID: 4, status: http.StatusNotFound},
		{name: "managed", id: "8", userID: 3, status: http.StatusNotFound},
		{name: "conflict", id: "7", userID: 3, err: apikey.ErrAPIKeyRotationConflict, status: http.StatusConflict},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &rotationHTTPRepository{
				apiKeyHandlerSecurityRepoStub: apiKeyHandlerSecurityRepoStub{keys: map[int64]*apikey.APIKey{
					7: {ID: 7, UserID: 3, Key: "sk-old", Name: "key", Status: apikey.StatusAPIKeyActive, QuotaUsed: 12},
					8: {ID: 8, UserID: 3, ManagedBy: &managed},
				}},
				err: test.err,
			}
			svc := apikey.NewAPIKeyService(repo, nil, nil, nil, nil, nil, &apikey.Options{})
			router := gin.New()
			group := router.Group("/api/v1")
			if test.userID != 0 {
				group.Use(func(c *gin.Context) {
					c.Set(authctx.ContextKeyUser, authctx.AuthSubject{UserID: test.userID})
				})
			}
			RegisterUserRoutes(group, NewAPIKeyHandler(svc, func(*routing.Group, *accessview.GroupCapacitySummary) *struct{} {
				return nil
			}))
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/keys/"+test.id+"/rotate", nil))
			require.Equal(t, test.status, rec.Code, rec.Body.String())
			if test.status == http.StatusOK {
				var result struct {
					Data struct {
						ID        int64   `json:"id"`
						Key       string  `json:"key"`
						QuotaUsed float64 `json:"quota_used"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
				require.Equal(t, int64(7), result.Data.ID)
				require.Equal(t, float64(12), result.Data.QuotaUsed)
				require.Regexp(t, `^sk-[0-9a-f]{64}$`, result.Data.Key)
				require.Equal(t, result.Data.Key, repo.keys[7].Key)
			} else {
				require.NotContains(t, rec.Body.String(), "sk-old")
				if test.err == nil {
					require.Zero(t, repo.calls)
				}
			}
		})
	}
}
