package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func newTestChannel(key string) *model.Channel {
	zero := 0
	return &model.Channel{
		Id:   1,
		Key:  key,
		Type: constant.ChannelTypeOpenAI,
		// AutoBan nil = false; Setting nil = empty; Models empty
		AutoBan: &zero,
	}
}

func TestSetupContextForSelectedChannel_BYOK_WithUpstreamKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	common.SetContextKey(c, constant.ContextKeyBYOKUpstreamKey, "sk-real-upstream-key")

	channel := newTestChannel(constant.ChannelKeyForwardSentinel)

	apiErr := SetupContextForSelectedChannel(c, channel, "gpt-4o-mini")
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if got := common.GetContextKeyString(c, constant.ContextKeyChannelKey); got != "sk-real-upstream-key" {
		t.Fatalf("ContextKeyChannelKey = %q, want sk-real-upstream-key", got)
	}
}

func TestSetupContextForSelectedChannel_BYOK_MissingUpstreamReturns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	// Note: no ContextKeyBYOKUpstreamKey set

	channel := newTestChannel(constant.ChannelKeyForwardSentinel)

	apiErr := SetupContextForSelectedChannel(c, channel, "gpt-4o-mini")
	if apiErr == nil {
		t.Fatalf("expected error, got nil")
	}
	if apiErr.StatusCode != 401 {
		t.Fatalf("status code = %d, want 401", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Error(), "this channel requires BYOK format: sk-<token>:<upstream-key>") {
		t.Fatalf("error message = %q, want documented BYOK message", apiErr.Error())
	}
}

func TestSetupContextForSelectedChannel_NonBYOK_IgnoresUpstreamContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	// Set the context key to verify a non-BYOK channel ignores it.
	common.SetContextKey(c, constant.ContextKeyBYOKUpstreamKey, "sk-some-upstream")

	channel := newTestChannel("sk-real-stored-channel-key")

	apiErr := SetupContextForSelectedChannel(c, channel, "gpt-4o-mini")
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if got := common.GetContextKeyString(c, constant.ContextKeyChannelKey); got != "sk-real-stored-channel-key" {
		t.Fatalf("ContextKeyChannelKey = %q, want stored channel key (non-BYOK should ignore upstream context)", got)
	}
}
