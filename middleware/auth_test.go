package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

func TestSplitBYOKKey(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantToken    string
		wantUpstream string
	}{
		{"no colon", "sk-abc123", "sk-abc123", ""},
		{"basic split", "sk-abc123:sk-real-openai", "sk-abc123", "sk-real-openai"},
		{"multiple colons preserved on right", "sk-abc123:weird:upstream:value", "sk-abc123", "weird:upstream:value"},
		{"empty right half", "sk-abc123:", "sk-abc123", ""},
		{"colon at start", ":upstream", "", "upstream"},
		{"empty string", "", "", ""},
		{"legacy token with dash", "sk-abc123-7-mygroup", "sk-abc123-7-mygroup", ""},
		{"BYOK with dashed token (channel id and group)", "sk-abc123-7-mygroup:sk-upstream", "sk-abc123-7-mygroup", "sk-upstream"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotToken, gotUpstream := splitBYOKKey(tc.raw)
			if gotToken != tc.wantToken || gotUpstream != tc.wantUpstream {
				t.Fatalf("splitBYOKKey(%q) = (%q, %q), want (%q, %q)",
					tc.raw, gotToken, gotUpstream, tc.wantToken, tc.wantUpstream)
			}
		})
	}
}

// TestResolveBYOKUpstreamFromHeaders covers the precedence chain used by the
// URL-prefix carrier when synthesizing the Authorization header.
func TestResolveBYOKUpstreamFromHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("authorization with colon: take right half", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Authorization", "Bearer sk-other-token:sk-real-upstream-key")
		if got := resolveBYOKUpstreamFromHeaders(c); got != "sk-real-upstream-key" {
			t.Fatalf("got %q, want sk-real-upstream-key", got)
		}
	})

	t.Run("authorization no colon: take bearer value", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Authorization", "Bearer sk-real-openai-key")
		if got := resolveBYOKUpstreamFromHeaders(c); got != "sk-real-openai-key" {
			t.Fatalf("got %q, want sk-real-openai-key", got)
		}
	})

	t.Run("x-api-key fallback", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("x-api-key", "anth-key")
		if got := resolveBYOKUpstreamFromHeaders(c); got != "anth-key" {
			t.Fatalf("got %q, want anth-key", got)
		}
	})

	t.Run("x-goog-api-key fallback", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.Header.Set("x-goog-api-key", "gemini-key")
		if got := resolveBYOKUpstreamFromHeaders(c); got != "gemini-key" {
			t.Fatalf("got %q, want gemini-key", got)
		}
	})

	t.Run("query ?key= fallback and consumed", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/v1beta/models/gemini-pro?key=real-gemini-key&other=keep", nil)
		if got := resolveBYOKUpstreamFromHeaders(c); got != "real-gemini-key" {
			t.Fatalf("got %q, want real-gemini-key", got)
		}
		if c.Request.URL.RawQuery == "" || strings.Contains(c.Request.URL.RawQuery, "key=") {
			t.Fatalf("?key= should be consumed; got RawQuery=%q", c.Request.URL.RawQuery)
		}
		if !strings.Contains(c.Request.URL.RawQuery, "other=keep") {
			t.Fatalf("non-consumed query params dropped: %q", c.Request.URL.RawQuery)
		}
	})

	t.Run("empty when no carriers present", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/", nil)
		if got := resolveBYOKUpstreamFromHeaders(c); got != "" {
			t.Fatalf("got %q, want empty", got)
		}
	})
}

// TestHandleBYOKURLRewrite_IntegrationStyle verifies the URL prefix carrier
// rewrites the request and synthesizes the canonical Authorization header
// before re-dispatching into the matching route. It covers every scenario
// listed in specs/channel-byok/spec.md for the URL-prefix requirement.
func TestHandleBYOKURLRewrite_IntegrationStyle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	build := func(register func(engine *gin.Engine, captured *struct {
		path string
		auth string
		raw  string
	})) (*gin.Engine, *struct {
		path string
		auth string
		raw  string
	}) {
		engine := gin.New()
		captured := &struct {
			path string
			auth string
			raw  string
		}{}
		register(engine, captured)
		engine.NoRoute(func(c *gin.Context) {
			if HandleBYOKURLRewrite(engine, c) {
				return
			}
			c.Status(404)
		})
		return engine, captured
	}

	t.Run("URL prefix with bearer-only upstream", func(t *testing.T) {
		engine, captured := build(func(engine *gin.Engine, captured *struct {
			path string
			auth string
			raw  string
		}) {
			engine.POST("/v1/chat/completions", func(c *gin.Context) {
				captured.path = c.Request.URL.Path
				captured.auth = c.Request.Header.Get("Authorization")
				c.Status(200)
			})
		})
		req := httptest.NewRequest("POST", "/sk-abc123/v1/chat/completions", nil)
		req.Header.Set("Authorization", "Bearer sk-real-openai-key")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("status=%d", w.Code)
		}
		if captured.path != "/v1/chat/completions" {
			t.Fatalf("path=%q", captured.path)
		}
		if captured.auth != "Bearer sk-abc123:sk-real-openai-key" {
			t.Fatalf("auth=%q", captured.auth)
		}
	})

	t.Run("URL prefix with no Authorization (no colon appended)", func(t *testing.T) {
		engine, captured := build(func(engine *gin.Engine, captured *struct {
			path string
			auth string
			raw  string
		}) {
			engine.POST("/v1/chat/completions", func(c *gin.Context) {
				captured.path = c.Request.URL.Path
				captured.auth = c.Request.Header.Get("Authorization")
				c.Status(200)
			})
		})
		req := httptest.NewRequest("POST", "/sk-abc123/v1/chat/completions", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("status=%d", w.Code)
		}
		if captured.auth != "Bearer sk-abc123" {
			t.Fatalf("auth=%q (should not contain colon)", captured.auth)
		}
	})

	t.Run("URL token wins when Authorization already has colon", func(t *testing.T) {
		engine, captured := build(func(engine *gin.Engine, captured *struct {
			path string
			auth string
			raw  string
		}) {
			engine.POST("/v1/chat/completions", func(c *gin.Context) {
				captured.auth = c.Request.Header.Get("Authorization")
				c.Status(200)
			})
		})
		req := httptest.NewRequest("POST", "/sk-abc123/v1/chat/completions", nil)
		req.Header.Set("Authorization", "Bearer sk-other-token:sk-real-upstream-key")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if captured.auth != "Bearer sk-abc123:sk-real-upstream-key" {
			t.Fatalf("auth=%q", captured.auth)
		}
	})

	t.Run("Gemini ?key= consumed", func(t *testing.T) {
		engine, captured := build(func(engine *gin.Engine, captured *struct {
			path string
			auth string
			raw  string
		}) {
			engine.GET("/v1beta/models/gemini-pro", func(c *gin.Context) {
				captured.path = c.Request.URL.Path
				captured.auth = c.Request.Header.Get("Authorization")
				captured.raw = c.Request.URL.RawQuery
				c.Status(200)
			})
		})
		req := httptest.NewRequest("GET", "/sk-abc123/v1beta/models/gemini-pro?key=real-gemini-key", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if captured.auth != "Bearer sk-abc123:real-gemini-key" {
			t.Fatalf("auth=%q", captured.auth)
		}
		if strings.Contains(captured.raw, "key=") {
			t.Fatalf("?key= not consumed: %q", captured.raw)
		}
	})

	t.Run("Path without /sk- prefix is not touched", func(t *testing.T) {
		hit := false
		engine, _ := build(func(engine *gin.Engine, _ *struct {
			path string
			auth string
			raw  string
		}) {
			engine.POST("/v1/chat/completions", func(c *gin.Context) {
				hit = true
				if c.Request.URL.Path != "/v1/chat/completions" {
					t.Fatalf("path mutated: %q", c.Request.URL.Path)
				}
				c.Status(200)
			})
		})
		req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if !hit {
			t.Fatalf("matched route did not fire")
		}
	})

	t.Run("URL prefix without trailing slash does not match", func(t *testing.T) {
		engine, _ := build(func(_ *gin.Engine, _ *struct {
			path string
			auth string
			raw  string
		}) {
		})
		req := httptest.NewRequest("POST", "/sk-abc123", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code != 404 {
			t.Fatalf("expected 404 fallback, got %d", w.Code)
		}
	})
}

// TestSplitBYOKKey_SetsContextWhenNonEmpty verifies the contract for setting
// the BYOK upstream key context — caller writes when upstream is non-empty,
// otherwise leaves the context untouched.
func TestSplitBYOKKey_ContextWriteContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("colon present: upstream context populated", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/", nil)
		token, upstream := splitBYOKKey("sk-abc:sk-up")
		if upstream != "" {
			common.SetContextKey(c, constant.ContextKeyBYOKUpstreamKey, upstream)
		}
		if token != "sk-abc" {
			t.Fatalf("token=%q", token)
		}
		if got := common.GetContextKeyString(c, constant.ContextKeyBYOKUpstreamKey); got != "sk-up" {
			t.Fatalf("ctx upstream=%q", got)
		}
	})

	t.Run("no colon: upstream context untouched", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/", nil)
		token, upstream := splitBYOKKey("sk-abc")
		if upstream != "" {
			common.SetContextKey(c, constant.ContextKeyBYOKUpstreamKey, upstream)
		}
		if token != "sk-abc" || upstream != "" {
			t.Fatalf("token=%q upstream=%q", token, upstream)
		}
		if got := common.GetContextKeyString(c, constant.ContextKeyBYOKUpstreamKey); got != "" {
			t.Fatalf("ctx upstream should be empty, got %q", got)
		}
	})
}
