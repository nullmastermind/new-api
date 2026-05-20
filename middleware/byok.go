package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// WrapBYOKURLRewrite wraps an http.Handler so that requests whose path starts
// with `/sk-<token>/` are rewritten BEFORE the inner handler sees them. The
// `sk-<token>` segment becomes the new-api token (synthesized into the
// Authorization header), and the existing Authorization / x-api-key /
// x-goog-api-key / `?key=` carriers contribute the upstream key per the
// precedence chain documented on HandleBYOKURLRewrite.
//
// This wrapper is the PRIMARY entry point for the URL-prefix BYOK carrier.
// It runs before any gin middleware (including response gzip), avoiding the
// streaming/gzip interaction bug where the NoRoute path leaves a dangling
// Content-Encoding: gzip header on raw SSE bodies.
func WrapBYOKURLRewrite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rewriteBYOKURLIfMatch(r)
		next.ServeHTTP(w, r)
	})
}

// rewriteBYOKURLIfMatch performs the /sk-<token>/ → / rewrite on the given
// request in place. If the path does not match the prefix, the request is
// left untouched.
func rewriteBYOKURLIfMatch(r *http.Request) {
	if r == nil || r.URL == nil {
		return
	}
	match := byokURLPrefixRegex.FindStringSubmatchIndex(r.URL.Path)
	if match == nil {
		return
	}
	tokenStart, tokenEnd := match[2], match[3]
	token := r.URL.Path[tokenStart:tokenEnd]
	rewrittenPath := r.URL.Path[tokenEnd:]
	if !strings.HasPrefix(rewrittenPath, "/") {
		rewrittenPath = "/" + rewrittenPath
	}

	upstream := resolveBYOKUpstreamFromHTTPRequest(r)

	var authValue string
	if upstream != "" {
		authValue = "Bearer " + token + ":" + upstream
	} else {
		authValue = "Bearer " + token
	}
	r.Header.Set("Authorization", authValue)
	r.Header.Del("x-api-key")
	r.Header.Del("x-goog-api-key")

	r.URL.Path = rewrittenPath
	r.URL.RawPath = ""
	if r.URL.RawQuery != "" {
		r.RequestURI = rewrittenPath + "?" + r.URL.RawQuery
	} else {
		r.RequestURI = rewrittenPath
	}
}

// resolveBYOKUpstreamFromHTTPRequest mirrors resolveBYOKUpstreamFromHeaders
// but operates on a plain *http.Request (no gin.Context required). The
// `?key=` query parameter is consumed when used (removed from the URL).
func resolveBYOKUpstreamFromHTTPRequest(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); auth != "" {
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
	if v := strings.TrimSpace(r.Header.Get("x-api-key")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.Header.Get("x-goog-api-key")); v != "" {
		return v
	}
	q := r.URL.Query()
	if v := q.Get("key"); v != "" {
		q.Del("key")
		r.URL.RawQuery = q.Encode()
		return v
	}
	return ""
}

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
