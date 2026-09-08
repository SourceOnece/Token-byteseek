//go:build unit

package admin

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCodexQualityHandlerRejectsInvalidAndUnconfirmedRequests(t *testing.T) {
	for _, body := range []string{`{}`, `{"account_ids":[1],"model":"gpt-6-astra","prompt":"test","keyword":"PASS"}`, `{"confirm_scheduling":true,"account_ids":[1,1],"model":"gpt-6-astra","prompt":"test","keyword":"PASS"}`} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
		(&AccountHandler{}).BatchCodexQualityTest(c)
		require.Equal(t, 400, w.Code)
	}
}

type qualityDeleteRepoStub struct {
	service.AccountRepository
	service.CodexQualityScheduleRepository
	deleted bool
	err     error
	ids     []int64
}

func (r *qualityDeleteRepoStub) DeleteQualitySchedule(_ context.Context, id int64) (bool, error) {
	r.ids = append(r.ids, id)
	return r.deleted, r.err
}

func TestCodexQualityDeleteRequiresConfirmationAndExactTarget(t *testing.T) {
	for _, tt := range []struct {
		name, id, body string
		deleted        bool
		err            error
		code, calls    int
	}{
		{"missing confirmation", "7", `{}`, true, nil, 400, 0},
		{"false confirmation", "7", `{"confirm_delete":false}`, true, nil, 400, 0},
		{"string confirmation", "7", `{"confirm_delete":"true"}`, true, nil, 400, 0},
		{"bad id", "0", `{"confirm_delete":true}`, true, nil, 400, 0},
		{"invalid id", "all", `{"confirm_delete":true}`, true, nil, 400, 0},
		{"confirmed", "7", `{"confirm_delete":true}`, true, nil, 200, 1},
		{"missing plan", "7", `{"confirm_delete":true}`, false, nil, 404, 1},
		{"db failure", "7", `{"confirm_delete":true}`, false, errors.New("secret-db-error"), 500, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &qualityDeleteRepoStub{deleted: tt.deleted, err: tt.err}
			s := service.NewAccountTestService(r, nil, nil, nil, nil, nil, nil, nil, nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "id", Value: tt.id}}
			c.Request = httptest.NewRequest("DELETE", "/", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			(&AccountHandler{accountTestService: s}).DeleteQualitySchedule(c)
			require.Equal(t, tt.code, w.Code)
			require.Len(t, r.ids, tt.calls)
			if tt.calls > 0 {
				require.Equal(t, int64(7), r.ids[0])
			}
			require.NotContains(t, w.Body.String(), "secret-db-error")
		})
	}
}

func TestCodexQualityHandlerRejectsUnavailableService(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"confirm_scheduling":true,"account_ids":[1],"model":"gpt-6-astra","prompt":"test","keyword":"PASS"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	(&AccountHandler{}).BatchCodexQualityTest(c)
	require.Equal(t, 503, w.Code)
}
