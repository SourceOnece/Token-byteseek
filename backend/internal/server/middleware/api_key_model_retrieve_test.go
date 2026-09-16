package middleware

import (
	"github.com/TokenFlux/TokenRouter/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestModelRetrieveRetainsCompositeIdentityWithoutBillingBypass(t *testing.T) {
	for _, path := range []string{"/v1/models/Prefix/vendor/model", "/models/Prefix/model"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", path, nil)
		key := &service.APIKey{ID: 1, IsComposite: true}
		resolved, err := resolveCompositeAPIKeyRequest(c, nil, key)
		require.NoError(t, err)
		require.Same(t, key, resolved)
		require.True(t, c.GetBool(compositeKeyNoGroupContextKey))
		require.True(t, isAPIKeyNonConsumingRequest("GET", path))
		require.False(t, isCompositeKeyBillingBypassEndpoint("GET", path))
		require.False(t, isCompositeKeyModelListEndpoint("POST", path))
	}
	require.False(t, isStandardModelRetrieveEndpoint("GET", "/v1/models-extra/private"))
}
