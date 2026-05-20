package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
)

// TestCheckBYOKTestKey_BYOKChannelMissingKeyFailsFast verifies that a BYOK
// channel under test with an empty/whitespace byok_test_key returns a non-nil
// error BEFORE any upstream call happens, and that the error carries the
// skip-retry option so service.ShouldDisableChannel returns false and the
// channel is NOT auto-banned by a missing-credential failure.
func TestCheckBYOKTestKey_BYOKChannelMissingKeyFailsFast(t *testing.T) {
	cases := []struct {
		name        string
		byokTestKey string
	}{
		{"absent (empty string)", ""},
		{"whitespace only", "   "},
		{"tab and newline only", "\t\n"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ch := &model.Channel{
				Id:   1,
				Type: constant.ChannelTypeOpenAI,
				Key:  constant.ChannelKeyForwardSentinel,
			}
			override, apiErr := checkBYOKTestKey(ch, tc.byokTestKey)
			if apiErr == nil {
				t.Fatalf("expected non-nil error for missing byok_test_key, got nil")
			}
			if override != "" {
				t.Fatalf("override = %q, want empty when error is returned", override)
			}
			if !types.IsSkipRetryError(apiErr) {
				t.Fatalf("error should carry skip-retry option so auto-ban is NOT triggered")
			}
			if apiErr.GetErrorCode() != types.ErrorCodeAccessDenied {
				t.Fatalf("error code = %v, want ErrorCodeAccessDenied", apiErr.GetErrorCode())
			}
			if got := apiErr.Error(); got != "BYOK channel requires test upstream key" {
				t.Fatalf("error message = %q, want %q", got, "BYOK channel requires test upstream key")
			}
		})
	}
}

// TestCheckBYOKTestKey_BYOKChannelWithKey verifies that a BYOK channel +
// non-empty byok_test_key returns the trimmed override key and nil error.
func TestCheckBYOKTestKey_BYOKChannelWithKey(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantOverride string
	}{
		{"plain key", "sk-real-test-key", "sk-real-test-key"},
		{"key with leading/trailing whitespace is trimmed",
			"  sk-real-test-key\n", "sk-real-test-key"},
		{"key with embedded colons preserved",
			"sk-real:upstream:weird", "sk-real:upstream:weird"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ch := &model.Channel{
				Id:   1,
				Type: constant.ChannelTypeOpenAI,
				Key:  constant.ChannelKeyForwardSentinel,
			}
			override, apiErr := checkBYOKTestKey(ch, tc.input)
			if apiErr != nil {
				t.Fatalf("unexpected error: %v", apiErr)
			}
			if override != tc.wantOverride {
				t.Fatalf("override = %q, want %q", override, tc.wantOverride)
			}
		})
	}
}

// TestCheckBYOKTestKey_BYOKMultiKeyChannelGate verifies that a BYOK multi-key
// channel (where the sentinel is one of several `\n`-separated lines) still
// gates the test on byok_test_key, mirroring the IsForwardKeyMode semantics.
func TestCheckBYOKTestKey_BYOKMultiKeyChannelGate(t *testing.T) {
	ch := &model.Channel{
		Id:   1,
		Type: constant.ChannelTypeOpenAI,
		Key:  "realkey1\n" + constant.ChannelKeyForwardSentinel + "\nrealkey3",
	}
	// empty key → fail fast
	if _, apiErr := checkBYOKTestKey(ch, ""); apiErr == nil {
		t.Fatalf("multi-key BYOK channel with empty byok_test_key should error")
	}
	// real key → returned as override
	override, apiErr := checkBYOKTestKey(ch, "sk-up")
	if apiErr != nil {
		t.Fatalf("unexpected error: %v", apiErr)
	}
	if override != "sk-up" {
		t.Fatalf("override = %q, want sk-up", override)
	}
}

// TestCheckBYOKTestKey_NonBYOKChannelIgnoresKey verifies that a non-BYOK
// channel returns empty override + nil error regardless of byok_test_key,
// signalling the caller to proceed with the stored channel key.
func TestCheckBYOKTestKey_NonBYOKChannelIgnoresKey(t *testing.T) {
	cases := []struct {
		name        string
		byokTestKey string
	}{
		{"with key present", "sk-some-test-key"},
		{"with whitespace", "   "},
		{"absent", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ch := &model.Channel{
				Id:   1,
				Type: constant.ChannelTypeOpenAI,
				Key:  "sk-real-stored-key", // NOT the sentinel
			}
			override, apiErr := checkBYOKTestKey(ch, tc.byokTestKey)
			if apiErr != nil {
				t.Fatalf("non-BYOK channel: expected nil error, got %v", apiErr)
			}
			if override != "" {
				t.Fatalf("non-BYOK channel: expected empty override, got %q", override)
			}
		})
	}
}

// TestCheckBYOKTestKey_NilChannel verifies the helper's defensive nil-channel
// handling: returns empty override + nil error (caller proceeds, downstream
// will surface a different error for the missing channel).
func TestCheckBYOKTestKey_NilChannel(t *testing.T) {
	override, apiErr := checkBYOKTestKey(nil, "sk-anything")
	if apiErr != nil {
		t.Fatalf("nil channel: expected nil error, got %v", apiErr)
	}
	if override != "" {
		t.Fatalf("nil channel: expected empty override, got %q", override)
	}
}

// TestCheckBYOKTestKey_SentinelAsSubstringNotByOK verifies that channels whose
// stored Key contains the sentinel only as a substring (not an exact line
// match) are NOT treated as BYOK and the byok_test_key is ignored —
// guarding the exact-line-match semantics of IsForwardKeyMode at the
// channel-test gate.
func TestCheckBYOKTestKey_SentinelAsSubstringNotByOK(t *testing.T) {
	ch := &model.Channel{
		Id:   1,
		Type: constant.ChannelTypeOpenAI,
		Key:  constant.ChannelKeyForwardSentinel + "_extra", // substring only
	}
	override, apiErr := checkBYOKTestKey(ch, "")
	if apiErr != nil {
		t.Fatalf("substring-sentinel channel must not gate on BYOK, got error: %v", apiErr)
	}
	if override != "" {
		t.Fatalf("override = %q, want empty (channel is not BYOK)", override)
	}
}
