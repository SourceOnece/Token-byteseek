//go:build unit

package admin

import (
	"net/http/httptest"
	"strings"
	"testing"

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

func TestCodexQualityHandlerRejectsUnavailableService(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"confirm_scheduling":true,"account_ids":[1],"model":"gpt-6-astra","prompt":"test","keyword":"PASS"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	(&AccountHandler{}).BatchCodexQualityTest(c)
	require.Equal(t, 503, w.Code)
}
