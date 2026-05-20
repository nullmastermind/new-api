package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

// TestExtractTokenAuthCarrier_AllCarriersConverge confirms that every supported
// auth carrier — Authorization Bearer/bearer, x-api-key on Claude paths,
// x-goog-api-key and ?key= on Gemini paths, mj-api-secret fallback, and the
// Sec-WebSocket-Protocol openai-insecure-api-key segment — funnels through a
// single convergence point and produces identical BYOK upstream-key context
// state. This guards the convergence-point assumption: any future refactor
// that breaks "all carriers eventually feed into Authorization before split"
// will be detected here.
//
// Each case carries `sk-FAKETOKEN:sk-real-upstream` via one carrier, runs
// extractTokenAuthCarrier (the function extracted from TokenAuth that performs
// all carrier-to-Authorization rewrites plus the BYOK colon split), and
// asserts:
//   - the parsed token half is "FAKETOKEN" (post sk- strip, pre `-` parts split)
//   - ContextKeyBYOKUpstreamKey is set to "sk-real-upstream"
func TestExtractTokenAuthCarrier_AllCarriersConverge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const wantToken = "FAKETOKEN"
	const wantUpstream = "sk-real-upstream"
	const carrierValue = "sk-FAKETOKEN:sk-real-upstream"

	cases := []struct {
		name      string
		method    string
		path      string
		setupReq  func(req *gin.Context)
		// optional override expectations for upstream (e.g. tokens with embedded :)
		expectToken    string
		expectUpstream string
	}{
		{
			name:   "Authorization Bearer (capitalized)",
			method: "POST",
			path:   "/v1/chat/completions",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "Bearer "+carrierValue)
			},
		},
		{
			name:   "Authorization bearer (lowercase)",
			method: "POST",
			path:   "/v1/chat/completions",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "bearer "+carrierValue)
			},
		},
		{
			name:   "x-api-key on Claude /v1/messages path",
			method: "POST",
			path:   "/v1/messages",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("x-api-key", carrierValue)
			},
		},
		{
			name:   "x-api-key on /v1/models path",
			method: "GET",
			path:   "/v1/models",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("x-api-key", carrierValue)
			},
		},
		{
			name:   "x-goog-api-key on Gemini /v1beta/models path",
			method: "POST",
			path:   "/v1beta/models/gemini-pro:generateContent",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("x-goog-api-key", carrierValue)
			},
		},
		{
			name:   "?key= query on Gemini /v1beta/models path",
			method: "POST",
			path:   "/v1beta/models/gemini-pro:generateContent?key=" + carrierValue,
			setupReq: func(_ *gin.Context) {
				// nothing — value lives in the URL
			},
		},
		{
			name:   "mj-api-secret fallback (no Authorization header)",
			method: "POST",
			path:   "/mj/submit/imagine",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("mj-api-secret", carrierValue)
			},
		},
		{
			name:   "mj-api-secret fallback (Authorization=midjourney-proxy sentinel)",
			method: "POST",
			path:   "/mj/submit/imagine",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "midjourney-proxy")
				c.Request.Header.Set("mj-api-secret", carrierValue)
			},
		},
		{
			name:   "WebSocket Sec-WebSocket-Protocol carrier",
			method: "GET",
			path:   "/v1/realtime",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("Sec-WebSocket-Protocol",
					"realtime, openai-insecure-api-key."+carrierValue+", openai-beta.realtime-v1")
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(tc.method, tc.path, nil)
			tc.setupReq(c)

			gotKey, gotParts := extractTokenAuthCarrier(c)

			expectToken := tc.expectToken
			if expectToken == "" {
				expectToken = wantToken
			}
			expectUpstream := tc.expectUpstream
			if expectUpstream == "" {
				expectUpstream = wantUpstream
			}

			if gotKey != expectToken {
				t.Fatalf("carrier %q: parsed token = %q, want %q (parts=%v)",
					tc.name, gotKey, expectToken, gotParts)
			}
			ctxUpstream := common.GetContextKeyString(c, constant.ContextKeyBYOKUpstreamKey)
			if ctxUpstream != expectUpstream {
				t.Fatalf("carrier %q: ContextKeyBYOKUpstreamKey = %q, want %q",
					tc.name, ctxUpstream, expectUpstream)
			}
		})
	}
}

// TestExtractTokenAuthCarrier_NoColon_NoUpstreamContext confirms that when no
// `:` is present in any carrier, ContextKeyBYOKUpstreamKey is NOT populated
// (legacy single-key behavior preserved byte-identically).
func TestExtractTokenAuthCarrier_NoColon_NoUpstreamContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name     string
		method   string
		path     string
		setupReq func(c *gin.Context)
	}{
		{
			name:   "Authorization Bearer legacy",
			method: "POST",
			path:   "/v1/chat/completions",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("Authorization", "Bearer sk-FAKETOKEN")
			},
		},
		{
			name:   "x-api-key legacy on Claude path",
			method: "POST",
			path:   "/v1/messages",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("x-api-key", "sk-FAKETOKEN")
			},
		},
		{
			name:   "x-goog-api-key legacy on Gemini path",
			method: "POST",
			path:   "/v1beta/models/gemini-pro:generateContent",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("x-goog-api-key", "sk-FAKETOKEN")
			},
		},
		{
			name:   "?key= legacy on Gemini path",
			method: "POST",
			path:   "/v1beta/models/gemini-pro:generateContent?key=sk-FAKETOKEN",
			setupReq: func(_ *gin.Context) {
			},
		},
		{
			name:   "mj-api-secret legacy",
			method: "POST",
			path:   "/mj/submit/imagine",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("mj-api-secret", "sk-FAKETOKEN")
			},
		},
		{
			name:   "WebSocket Sec-WebSocket-Protocol legacy",
			method: "GET",
			path:   "/v1/realtime",
			setupReq: func(c *gin.Context) {
				c.Request.Header.Set("Sec-WebSocket-Protocol",
					"realtime, openai-insecure-api-key.sk-FAKETOKEN, openai-beta.realtime-v1")
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(tc.method, tc.path, nil)
			tc.setupReq(c)

			gotKey, _ := extractTokenAuthCarrier(c)
			if gotKey != "FAKETOKEN" {
				t.Fatalf("carrier %q: parsed token = %q, want FAKETOKEN", tc.name, gotKey)
			}
			ctxUpstream := common.GetContextKeyString(c, constant.ContextKeyBYOKUpstreamKey)
			if ctxUpstream != "" {
				t.Fatalf("carrier %q: ContextKeyBYOKUpstreamKey should be unset, got %q",
					tc.name, ctxUpstream)
			}
		})
	}
}

// TestExtractTokenAuthCarrier_PreservesUpstreamColons confirms the right half
// of the split is preserved verbatim — including embedded `:` characters —
// across every carrier. SplitN(":", 2) semantics, not full split.
func TestExtractTokenAuthCarrier_PreservesUpstreamColons(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const carrierValue = "sk-FAKETOKEN:weird:upstream:value"
	const wantUpstream = "weird:upstream:value"

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request.Header.Set("Authorization", "Bearer "+carrierValue)

	gotKey, _ := extractTokenAuthCarrier(c)
	if gotKey != "FAKETOKEN" {
		t.Fatalf("parsed token = %q, want FAKETOKEN", gotKey)
	}
	if got := common.GetContextKeyString(c, constant.ContextKeyBYOKUpstreamKey); got != wantUpstream {
		t.Fatalf("upstream = %q, want %q", got, wantUpstream)
	}
}

// TestParseAuthCarrierValue_BYOKSplit confirms the shared carrier parser
// — used by both Authorization and mj-api-secret branches in
// extractTokenAuthCarrier and reused by TokenAuthReadOnly's BYOK discard —
// produces identical token-parts shape and preserves the upstream half.
func TestParseAuthCarrierValue_BYOKSplit(t *testing.T) {
	cases := []struct {
		name         string
		raw          string
		wantKey      string
		wantParts    []string
		wantUpstream string
	}{
		{
			name:         "BYOK with sk- token",
			raw:          "sk-FAKETOKEN:sk-real-upstream",
			wantKey:      "FAKETOKEN",
			wantParts:    []string{"FAKETOKEN"},
			wantUpstream: "sk-real-upstream",
		},
		{
			name:         "BYOK preserves embedded colons in upstream",
			raw:          "sk-FAKETOKEN:weird:upstream:value",
			wantKey:      "FAKETOKEN",
			wantParts:    []string{"FAKETOKEN"},
			wantUpstream: "weird:upstream:value",
		},
		{
			name:         "legacy single-key, no colon",
			raw:          "sk-FAKETOKEN",
			wantKey:      "FAKETOKEN",
			wantParts:    []string{"FAKETOKEN"},
			wantUpstream: "",
		},
		{
			name:         "legacy single-key with channel-id suffix",
			raw:          "sk-FAKETOKEN-42",
			wantKey:      "FAKETOKEN",
			wantParts:    []string{"FAKETOKEN", "42"},
			wantUpstream: "",
		},
		{
			name:         "empty input",
			raw:          "",
			wantKey:      "",
			wantParts:    []string{""},
			wantUpstream: "",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gotKey, gotParts, gotUpstream := parseAuthCarrierValue(tc.raw)
			if gotKey != tc.wantKey {
				t.Fatalf("key = %q, want %q", gotKey, tc.wantKey)
			}
			if gotUpstream != tc.wantUpstream {
				t.Fatalf("upstream = %q, want %q", gotUpstream, tc.wantUpstream)
			}
			if len(gotParts) != len(tc.wantParts) {
				t.Fatalf("parts = %v, want %v", gotParts, tc.wantParts)
			}
			for i, p := range tc.wantParts {
				if gotParts[i] != p {
					t.Fatalf("parts[%d] = %q, want %q", i, gotParts[i], p)
				}
			}
		})
	}
}

// TestTokenAuthReadOnly_BYOKSplit_RegressionCoverage guards the FIX for
// TokenAuthReadOnly: a BYOK-formatted carrier (`sk-<token>:<upstream>`)
// must yield the same parsed token half as the legacy single-key path
// so the downstream model.GetTokenByKey lookup succeeds. Before the fix
// the raw value `sk-FAKETOKEN:sk-upstream` would parse to key
// `FAKETOKEN:sk` and 401 the request.
//
// We assert the parser slice — splitBYOKKey + sk- strip + `-` split — that
// the (DB-touching) TokenAuthReadOnly middleware uses, NOT the full
// middleware (which would require a DB). The middleware's call sequence
// is `key, _ = splitBYOKKey(key); key = TrimPrefix("sk-"); parts =
// Split("-"); key = parts[0]` — exercised here directly.
func TestTokenAuthReadOnly_BYOKSplit_RegressionCoverage(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantKey string
	}{
		{name: "BYOK formatted carrier", raw: "sk-FAKETOKEN:sk-real-upstream", wantKey: "FAKETOKEN"},
		{name: "BYOK with embedded colons in upstream", raw: "sk-FAKETOKEN:up:stream", wantKey: "FAKETOKEN"},
		{name: "legacy single key (no colon)", raw: "sk-FAKETOKEN", wantKey: "FAKETOKEN"},
		{name: "legacy with channel-id suffix", raw: "sk-FAKETOKEN-42", wantKey: "FAKETOKEN"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			key := tc.raw
			// Mirror exactly the (post-fix) parsing block from TokenAuthReadOnly.
			key, _ = splitBYOKKey(key)
			key = strings.TrimPrefix(key, "sk-")
			parts := strings.Split(key, "-")
			key = parts[0]
			if key != tc.wantKey {
				t.Fatalf("parsed key = %q, want %q", key, tc.wantKey)
			}
		})
	}
}

// TestBYOKURLPrefixRegex_LengthBound guards the {1,128} upper bound on
// the URL-prefix capture group: pathological-length tokens must NOT
// match (request falls through to the 404 path instead of being
// rewritten); reasonable-length tokens must still match.
func TestBYOKURLPrefixRegex_LengthBound(t *testing.T) {
	cases := []struct {
		name      string
		path      string
		wantMatch bool
	}{
		{name: "short token matches", path: "/sk-abc/v1/chat", wantMatch: true},
		{name: "48-char inner matches (real-world size)", path: "/sk-" + strings.Repeat("a", 48) + "/v1/chat", wantMatch: true},
		{name: "128-char inner matches at boundary", path: "/sk-" + strings.Repeat("a", 128) + "/v1/chat", wantMatch: true},
		{name: "129-char inner does NOT match (over bound)", path: "/sk-" + strings.Repeat("a", 129) + "/v1/chat", wantMatch: false},
		{name: "200-char inner does NOT match", path: "/sk-" + strings.Repeat("a", 200) + "/v1/chat", wantMatch: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := byokURLPrefixRegex.FindStringSubmatchIndex(tc.path) != nil
			if got != tc.wantMatch {
				t.Fatalf("regex match for %q = %v, want %v", tc.path, got, tc.wantMatch)
			}
		})
	}
}
