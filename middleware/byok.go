package middleware

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// byokURLPrefixRegex matches a leading `/sk-<token>/` segment on a request
// path. The trailing `/` is REQUIRED — bare `/sk-...` paths (no further
// segment) do NOT match, so they fall through to the existing 404 path
// instead of being rewritten. The `{1,128}` upper bound is defense-in-depth
// against pathological-length tokens reaching the rewrite path; real
// new-api tokens are ~48 chars so 128 is generous headroom.
var byokURLPrefixRegex = regexp.MustCompile(`^/(sk-[A-Za-z0-9_-]{1,128})/`)

// HandleBYOKURLRewrite implements the URL-prefix carrier for BYOK forwarding.
// When a request path matches `^/sk-<token>/`, the `sk-<token>` segment is
// extracted as the new-api token, an upstream key is resolved from existing
// auth carriers, and a synthetic `Authorization: Bearer sk-<token>:<upstream>`
// header is written before re-dispatching the request through Gin's normal
// routing via engine.HandleContext.
//
// Upstream-key precedence (per design):
//  1. The portion after the first `:` in the existing `Authorization` header
//     (URL token wins — discards the left half of the existing Authorization).
//  2. The raw bearer value of `Authorization` if it has no `:`.
//  3. `x-api-key` header.
//  4. `x-goog-api-key` header.
//  5. `?key=` query parameter (consumed — removed so re-dispatch doesn't
//     re-process it).
//  6. Empty (the distributor will return 401 for BYOK channels).
//
// Returns true if the request was rewritten and re-dispatched (caller must
// abort the current frame); false if the path did not match the prefix and
// the caller should continue with its normal handler.
func HandleBYOKURLRewrite(engine *gin.Engine, c *gin.Context) bool {
	if engine == nil || c == nil || c.Request == nil {
		return false
	}
	path := c.Request.URL.Path
	match := byokURLPrefixRegex.FindStringSubmatchIndex(path)
	if match == nil {
		return false
	}
	// match[2]..match[3] = capture group 1 (`sk-<token>` without the leading /)
	tokenStart, tokenEnd := match[2], match[3]
	token := path[tokenStart:tokenEnd]
	// Trim the `/sk-<token>` segment from the path, preserving the trailing `/`.
	rewrittenPath := path[tokenEnd:]
	if !strings.HasPrefix(rewrittenPath, "/") {
		rewrittenPath = "/" + rewrittenPath
	}

	upstream := resolveBYOKUpstreamFromHeaders(c)

	// Build the synthetic Authorization header. If the resolved upstream is
	// empty we still write a bearer-only value so the downstream TokenAuth
	// path can authenticate; the distributor will 401 if the target channel
	// requires BYOK.
	var authValue string
	if upstream != "" {
		authValue = "Bearer " + token + ":" + upstream
	} else {
		authValue = "Bearer " + token
	}
	c.Request.Header.Set("Authorization", authValue)
	// Clear all alternate auth carriers so the rewritten Authorization is
	// the single source of truth on re-dispatch — regardless of which
	// carrier supplied the upstream value above.
	c.Request.Header.Del("x-api-key")
	c.Request.Header.Del("x-goog-api-key")

	// Rewrite the path / RequestURI for downstream routing and re-dispatch.
	c.Request.URL.Path = rewrittenPath
	c.Request.URL.RawPath = ""
	// Preserve the query string when reconstructing RequestURI.
	if c.Request.URL.RawQuery != "" {
		c.Request.RequestURI = rewrittenPath + "?" + c.Request.URL.RawQuery
	} else {
		c.Request.RequestURI = rewrittenPath
	}

	engine.HandleContext(c)
	c.Abort()
	return true
}

// resolveBYOKUpstreamFromHeaders extracts the upstream key portion from the
// existing auth carriers, per the precedence chain documented on
// HandleBYOKURLRewrite. The `?key=` query parameter is consumed when used
// (removed from the URL) so the rewritten request does not see it twice.
func resolveBYOKUpstreamFromHeaders(c *gin.Context) string {
	// 1 & 2: Authorization header — split on first `:`, take right half if
	// present, else use the bearer value as-is.
	if auth := c.Request.Header.Get("Authorization"); auth != "" {
		raw := auth
		if strings.HasPrefix(raw, "Bearer ") || strings.HasPrefix(raw, "bearer ") {
			raw = strings.TrimSpace(raw[7:])
		}
		if idx := strings.IndexByte(raw, ':'); idx >= 0 {
			return raw[idx+1:]
		}
		if raw != "" {
			return raw
		}
	}
	// 3: x-api-key header (Claude carrier).
	if v := strings.TrimSpace(c.Request.Header.Get("x-api-key")); v != "" {
		return v
	}
	// 4: x-goog-api-key header (Gemini carrier).
	if v := strings.TrimSpace(c.Request.Header.Get("x-goog-api-key")); v != "" {
		return v
	}
	// 5: ?key= query parameter (Gemini carrier). Consume it so the
	// re-dispatched request does not re-process it.
	if v := c.Query("key"); v != "" {
		q := c.Request.URL.Query()
		q.Del("key")
		c.Request.URL.RawQuery = q.Encode()
		return v
	}
	return ""
}
