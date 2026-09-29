package httpapi

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/TokenFlux/TokenRouter/internal/upstream/openai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIResponsesRejectedFieldRetryStateForRequestAllowsSameTransformAcrossProviders(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	initialBody := []byte(`{"model":"gpt-5.5","truncation":"auto"}`)
	retryBody := []byte(`{"model":"gpt-5.5"}`)

	providerA := openAIResponsesRejectedFieldRetryStateForRequest(c, initialBody)
	require.True(t, providerA.Allow(retryBody))
	require.False(t, providerA.Allow(retryBody), "one provider must not repeat the same transform")

	providerB := openAIResponsesRejectedFieldRetryStateForRequest(c, initialBody)
	require.NotSame(t, providerA, providerB)
	require.Same(t, providerA.Budget(), providerB.Budget())
	require.True(t, providerB.Allow(retryBody), "a failover provider must be allowed to apply the same transform")
}

func TestOpenAIResponsesRejectedFieldRetryStateForRequestSharesBoundedBudgetAcrossProviders(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	for attempt := 0; attempt < openai.MaxResponsesRejectedFieldRetries; attempt++ {
		state := openAIResponsesRejectedFieldRetryStateForRequest(c, []byte(fmt.Sprintf(`{"provider":%d}`, attempt)))
		require.True(t, state.Allow([]byte(`{"same":"retry"}`)))
	}
	overflow := openAIResponsesRejectedFieldRetryStateForRequest(c, []byte(`{"provider":"overflow"}`))
	require.False(t, overflow.Allow([]byte(`{"new":"retry"}`)))
}
