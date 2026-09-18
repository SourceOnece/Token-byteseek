package handler

import (
	"context"
	"encoding/json"
	"github.com/TokenFlux/TokenRouter/internal/pkg/pagination"
	"github.com/TokenFlux/TokenRouter/internal/server/middleware"
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

type redeemHistoryRepo struct {
	service.RedeemCodeRepository
	user        int64
	legacyLimit int
	params      pagination.PaginationParams
}

func (r *redeemHistoryRepo) ListByUser(_ context.Context, user int64, limit int) ([]service.RedeemCode, error) {
	r.user, r.legacyLimit = user, limit
	return []service.RedeemCode{}, nil
}
func (r *redeemHistoryRepo) ListByUserPaginated(_ context.Context, user int64, p pagination.PaginationParams, kind string) ([]service.RedeemCode, *pagination.PaginationResult, error) {
	r.user, r.params = user, p
	return []service.RedeemCode{{ID: 3, Code: "example", Type: "subscription", Value: 1}}, &pagination.PaginationResult{Total: 35}, nil
}

// 旧客户端仍获得数组；新分页始终使用当前登录用户且保留套餐历史，不接受用户 ID 越权。
func TestRedeemHistoryPaginationCompatibility(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, query := range []string{"", "?page=2&page_size=10&user_id=999", "?page_size=200", "?page=0", "?page=-1", "?page=x", "?page_size=-5", "?page=9223372036854775807"} {
		t.Run(query, func(t *testing.T) {
			repo := &redeemHistoryRepo{}
			h := NewRedeemHandler(service.NewRedeemService(repo, nil, nil, nil, nil, nil, nil, nil))
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/redeem/history"+query, nil)
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 17})
			h.GetHistory(c)
			if query != "" && query != "?page=2&page_size=10&user_id=999" && query != "?page_size=200" {
				require.Equal(t, http.StatusBadRequest, rec.Code)
				require.Zero(t, repo.user)
				return
			}
			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, int64(17), repo.user)
			var envelope struct {
				Data json.RawMessage `json:"data"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
			if query == "" {
				require.JSONEq(t, `[]`, string(envelope.Data))
				require.Equal(t, 25, repo.legacyLimit)
			} else {
				var page struct {
					Items []service.RedeemCode `json:"items"`
					Total int                  `json:"total"`
					Page  int                  `json:"page"`
					Size  int                  `json:"page_size"`
				}
				require.NoError(t, json.Unmarshal(envelope.Data, &page))
				require.Equal(t, 35, page.Total)
				require.Equal(t, "subscription", page.Items[0].Type)
				require.Equal(t, repo.params.PageSize, page.Size)
				if query == "?page_size=200" {
					require.Equal(t, 100, page.Size)
				} else {
					require.Equal(t, 2, page.Page)
				}
			}
		})
	}
}
