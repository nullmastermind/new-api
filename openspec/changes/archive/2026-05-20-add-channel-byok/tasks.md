## 1. Constants and helpers

- [x] 1.1 Add `ContextKeyBYOKUpstreamKey` to `constant/context_key.go`
- [x] 1.2 Define `ChannelKeyForwardSentinel = "$FORWARD_KEY"` in `constant/channel.go` (create file if absent) and export
- [x] 1.3 Add `Channel.IsForwardKeyMode()` helper in `model/channel.go` that returns true when any `\n`-trimmed line of `channel.Key` equals the sentinel ← (verify: exact-line match semantics, multi-key & single-key both detected, substring not matched)

## 2. Auth middleware — colon split across all carriers

- [x] 2.1 In `middleware/auth.go:TokenAuth`, extract a small helper `splitBYOKKey(raw string) (newApiPart, upstreamPart string)` that performs `strings.SplitN(raw, ":", 2)` on the post-`sk-` value
- [x] 2.2 Apply the helper to the `Authorization` (Bearer/bearer) carrier path before the existing `-` split
- [x] 2.3 Apply the helper to the `mj-api-secret` fallback carrier path
- [x] 2.4 Apply the helper to the `Sec-WebSocket-Protocol` `openai-insecure-api-key.<value>` segment before the synthetic `Authorization` rewrite
- [x] 2.5 Apply the helper to the Claude-path `x-api-key` rewrite (before it becomes the synthetic `Authorization`)
- [x] 2.6 Apply the helper to the Gemini-path `?key=` and `x-goog-api-key` rewrites (before they become synthetic `Authorization`)
- [x] 2.7 Store the upstream half in the gin context via `common.SetContextKey(c, constant.ContextKeyBYOKUpstreamKey, upstreamPart)` when non-empty

## 3. Distributor — channel-key swap and 401

- [x] 3.1 In `middleware/distributor.go:SetupContextForSelectedChannel`, after `channel.GetNextEnabledKey()` and before `common.SetContextKey(c, ContextKeyChannelKey, key)`, branch on `channel.IsForwardKeyMode()`
- [x] 3.2 When BYOK: read `ContextKeyBYOKUpstreamKey` from the context. If empty, return `types.NewError(...)` with HTTP 401 and the exact message `"this channel requires BYOK format: sk-<token>:<upstream-key>"` (use `types.ErrOptionWithSkipRetry()` so the retry layer doesn't burn through other channels)
- [x] 3.3 When BYOK and upstream key present: pass the upstream key into the existing `common.SetContextKey(c, ContextKeyChannelKey, ...)` call so `info.ApiKey` is populated transparently
- [x] 3.4 When non-BYOK: existing behavior, no change

## 4. URL-prefix carrier — engine.NoRoute rewrite

- [x] 4.1 Identify the gin engine setup point (`main.go` or `router/main.go`) where routes are mounted and where `engine.NoRoute` can be registered without race with other NoRoute uses
- [x] 4.2 Implement a NoRoute handler that:
  - regex-matches `^/sk-[A-Za-z0-9_-]+/` against `c.Request.URL.Path`
  - returns to the default 404 path if no match
  - on match: extracts the `sk-<token>` segment, trims it from the path (preserving trailing slash semantics)
  - resolves the upstream key with the precedence chain: split of existing `Authorization` after first `:` → bearer value (no `:`) → `x-api-key` → `x-goog-api-key` → `?key=` query → empty
  - sets `c.Request.Header.Set("Authorization", "Bearer sk-<token>" + (":" + upstream if upstream non-empty else ""))`
  - clears consumed source headers / query param to avoid double-processing on re-dispatch
  - calls `engine.HandleContext(c)` and then `c.Abort()`
- [x] 4.3 Ensure the NoRoute handler is mounted exactly once, after all route groups are registered

## 5. Channel test — BYOK test key

- [x] 5.1 Add optional `byok_test_key string` field to the test request DTO consumed by `controller/channel-test.go`
- [x] 5.2 In `controller/channel-test.go`, after `info.InitChannelMeta`, detect BYOK channel via `channel.IsForwardKeyMode()`. If BYOK and the request's `byok_test_key` is non-empty, override `info.ApiKey` with that value
- [x] 5.3 If BYOK and `byok_test_key` is empty, short-circuit with a clear test error before any upstream call (do NOT trigger auto-ban paths)
- [x] 5.4 Non-BYOK channels: ignore `byok_test_key` entirely

## 6. Frontend default theme (`web/default/`)

- [x] 6.1 In `src/features/channels/components/drawers/channel-mutate-drawer.tsx`, add a helper line under the Key textarea field
- [x] 6.2 Add a clipboard-copy handler for the sentinel
- [x] 6.3 In `src/features/channels/components/dialogs/channel-test-dialog.tsx`, detect BYOK (channel.Key contains a line equal to the sentinel) and render an "Upstream test key (BYOK)" input that maps to the `byok_test_key` request field
- [x] 6.4 Add new i18n keys in `src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`

## 7. Frontend classic theme (`web/classic/`)

- [x] 7.1 In `src/components/table/channels/modals/EditChannelModal.jsx`, add the equivalent helper line + copy affordance under the Key field
- [x] 7.2 In the existing channel test dialog component for this theme, add the equivalent BYOK test-key input, gated by sentinel detection
- [x] 7.3 Add new i18n keys in `src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`

## 8. Tests

- [x] 8.1 Add `middleware/auth_test.go` cases for: `sk-token:upstream` parses both halves, `sk-token` alone leaves upstream empty, `sk-token:weird:upstream:value` keeps the full right half, all five carriers (`Authorization`, `x-api-key`, `x-goog-api-key`, `mj-api-secret`, WebSocket subprotocol) produce identical upstream-key context state — additionally covered by `middleware/auth_carriers_test.go::TestExtractTokenAuthCarrier_AllCarriersConverge` which drives the consolidated `extractTokenAuthCarrier` convergence function for every carrier (Bearer + bearer, x-api-key on `/v1/messages` and `/v1/models`, x-goog-api-key, `?key=`, mj-api-secret with and without `midjourney-proxy` Authorization sentinel, and Sec-WebSocket-Protocol) and asserts identical `(token, ContextKeyBYOKUpstreamKey)` state across all of them.
- [x] 8.2 Add `middleware/distributor_test.go` (or extend existing test file) covering: BYOK channel with upstream key swaps `ContextKeyChannelKey`; BYOK channel without upstream key returns 401 with documented message; non-BYOK channel ignores the upstream-key context
- [x] 8.3 Add `model/channel_test.go` (or extend existing) for `IsForwardKeyMode()`: single-key sentinel, multi-key sentinel-mixed, single real key (no), real key with sentinel as substring (no), empty key (no)
- [x] 8.4 Add an integration-style test for the URL rewrite: `/sk-XXX/v1/chat/completions` with `Authorization: Bearer sk-upstream` produces the canonical `Authorization: Bearer sk-XXX:sk-upstream` at the rewritten frame; `/v1/chat/completions` (no prefix) is not touched

## 9. Build, lint, license headers

- [x] 9.1 `go build ./...` succeeds with no new warnings
- [x] 9.2 `go test ./middleware/... ./model/... ./controller/...` succeeds
- [x] 9.3 Any new `.go` files contain the AGPL-3.0 license header in the project's existing format
- [x] 9.4 `bun run eslint` in `web/default/` and `web/classic/` succeeds
