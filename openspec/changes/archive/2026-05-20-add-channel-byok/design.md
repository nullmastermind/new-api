## Context

new-api currently requires `channel.Key` to be a real upstream credential stored in the database. All adapters consume that value via `info.ApiKey` (populated from `ContextKeyChannelKey` set by the distributor in `middleware/distributor.go:382`). The auth flow in `middleware/auth.go:TokenAuth` validates the `Authorization`/`x-api-key`/`x-goog-api-key`/`mj-api-secret` carriers as new-api tokens — clients have no way to pass their own upstream key, and the existing Header Override mechanism cannot help because it explicitly blocks `authorization`/`x-api-key`/`x-goog-api-key` from passthrough (`relay/channel/api_request.go:50-77`).

Conversation-level discovery confirmed three integration realities that shape the design:

1. The token carrier already supports a multi-segment form `sk-<token>-<channelid>-<group>` using `-` as separator (`auth.go:328-330`). Any new format must coexist with this without ambiguity.
2. Every adapter calls `header.Set("Authorization", "Bearer "+info.ApiKey)` (or provider-equivalent: `api-key`, `x-api-key`, `x-goog-api-key`, signed `Authorization`). If `info.ApiKey` is transparent at the distributor boundary, no adapter needs to change.
3. The `engine.NoRoute` + `engine.HandleContext` pattern in Gin allows path rewriting before route matching without duplicating route registrations.

## Goals / Non-Goals

**Goals:**
- Zero-schema-change channel-level opt-in for BYOK forwarding.
- Two equivalent client carrier formats (header `:` split, URL `/sk-<token>/` prefix) so clients with restricted base-URL-only configs still work.
- Transparent integration: adapters, quota, billing, logging, retry, auto-ban, and multi-key paths remain untouched.
- Operationally safe channel test for BYOK channels via admin-supplied temporary upstream key.
- Hard-fail (HTTP 401) when a BYOK channel receives a request without an upstream key, rather than leaking a dummy key to the upstream.

**Non-Goals:**
- A per-channel boolean toggle column or new settings field. The sentinel string in `channel.Key` IS the toggle.
- BYOK ergonomics for composite-key channels (AWS `access|secret|region`, Vertex JSON, Xunfei `appid|secret|key`, Volcengine `appid|token`). The mechanism works for them transparently but UX is not optimized — UI hint will state the limitation.
- Hashing, audit logging, or rotation of upstream keys. The upstream key lives only in-memory for one request lifecycle. Existing `RelayInfo.String` already masks `ApiKey`.
- Changes to `channel.Key NOT NULL` validation in `controller/channel.go:464-467` — the sentinel string satisfies it.
- Changes to the existing `sk-<token>-<channelid>-<group>` parsing behavior.

## Decisions

### Decision 1: Sentinel value `$FORWARD_KEY`

**Choice:** Literal exact-match string `$FORWARD_KEY`.

**Rationale:** `$` is not a legal character in any real OpenAI/Anthropic/Gemini/Azure-style key the operator might paste by accident, eliminating collision risk. Treating the channel.Key column itself as the toggle avoids schema changes, migration, and UI restructuring. Operators who type the sentinel are making a deliberate, visible choice in the same field they'd otherwise type a real key.

**Alternatives considered:**
- `FORWARD_KEY` without `$` — rejected: an operator could conceivably have a real provider key with that exact value, leading to silent BYOK mis-activation.
- Separate `byok_enabled` column or `ChannelSettings.ForwardKey bool` — rejected: requires schema migration and UI toggle plumbing across two frontend themes for a feature whose semantics are already perfectly expressed by "the stored key is not a real key".

### Decision 2: Header carrier — first-`:` split

**Choice:** In `TokenAuth`, before the existing `sk-` prefix strip and `-` split, perform `strings.SplitN(rawKey, ":", 2)`. The left half is the new-api token (continues through the existing parse). The right half is stored in `c.Set(ContextKeyBYOKUpstreamKey, ...)`.

**Rationale:** `:` is not used anywhere in the current carrier format, so the split is unambiguous. `SplitN(..., 2)` preserves any `:` characters inside the upstream key (e.g., provider keys that embed `:`). The left half is then fed unchanged into the existing parser, so all existing behaviors (sk- strip, `-` split for channel-id/group, `mj-api-secret` fallback) keep working byte-for-byte.

**Applies to all carriers:** `Authorization` (Bearer/bearer), `x-api-key` (claude paths), `x-goog-api-key` (gemini), `mj-api-secret`, `Sec-WebSocket-Protocol`'s `openai-insecure-api-key.<key>` segment, and gemini `?key=` query parameter.

**Alternatives considered:**
- Different separator (`|`, `#`, `;`) — rejected: `|` is already used in composite keys (`appid|secret`), `#` is fragile in URLs, `;` looks accidental.
- Two separate headers (e.g., add an `x-upstream-key` header) — rejected: many clients can only configure base URL + API key, not extra headers; this is the exact constraint that motivates the URL carrier too.

### Decision 3: URL carrier — `engine.NoRoute` rewrite

**Choice:** Register an `engine.NoRoute` handler that:

1. Matches `^/sk-[A-Za-z0-9_-]+/` against `c.Request.URL.Path`.
2. If matched, extracts the `sk-<token>` segment and trims it from `URL.Path`.
3. Synthesizes a new `Authorization: Bearer sk-<token>:<upstream>` header value where `<upstream>` is sourced (in precedence order) from:
   a. The part after the first `:` in the original `Authorization` header (Q7 — URL token wins, take only the right half).
   b. The original `Authorization` header's raw value after stripping `Bearer ` (if it contained no `:`).
   c. The original `x-api-key` header value.
   d. The original `x-goog-api-key` header value or `?key=` query parameter (gemini).
   e. Empty string (the distributor will then 401 if the channel is BYOK).
4. Writes the synthesized header back via `c.Request.Header.Set("Authorization", ...)` and clears the consumed source headers / query parameter to avoid double-processing.
5. Calls `engine.HandleContext(c)` to re-dispatch into the rewritten path, then `c.Abort()` on the NoRoute frame.

**Rationale:** `NoRoute` runs only when no other route matched, so the rewrite never accidentally hijacks an existing route. `HandleContext` re-enters Gin's routing with the modified request, letting all middleware (TokenAuth, ModelRequestRateLimit, Distribute) run with the canonical carrier — no code duplication.

**Alternatives considered:**
- Registering all relay routes a second time under a `/:nakey/v1/...` group with a custom middleware — rejected: tedious, error-prone, and easy to drift between the two mounts when new routes are added.
- A top-level engine `Use(...)` middleware that runs before routing — rejected: Gin matches the route first, then runs the middleware chain; a top-level `Use` doesn't get a chance to rewrite before matching.

### Decision 4: Distributor enforcement and key swap

**Choice:** In `middleware/distributor.go:SetupContextForSelectedChannel`, immediately after the existing `channel.GetNextEnabledKey()` call, branch on `channel.IsForwardKeyMode()`:

- If false → set `ContextKeyChannelKey` to the picked key (existing behavior).
- If true → read `ContextKeyBYOKUpstreamKey` from the gin context.
  - Non-empty → set `ContextKeyChannelKey` to the upstream key. `info.ApiKey` populated by `GenRelayInfo` then carries the upstream key transparently to every adapter.
  - Empty → return `types.NewError(...)` with HTTP 401 and the documented message: `"this channel requires BYOK format: sk-<token>:<upstream-key>"`.

**Rationale:** The single mutation point at the distributor boundary makes the change auditable and keeps adapters oblivious. The 401 response is consistent with how new-api signals other auth-level failures (TokenAuth uses `abortWithOpenAiMessage(c, http.StatusUnauthorized, ...)`).

### Decision 5: `IsForwardKeyMode()` semantics for multi-key

**Choice:** A channel is in BYOK mode if **any** `\n`-separated line of `channel.Key` (after `strings.TrimSpace`) is exactly equal to the sentinel string.

**Rationale:** Multi-key channels store keys joined by `\n` (`controller/channel.go:940-970`). Per Q4, mixing sentinel with real keys is allowed but channel-wide opts into BYOK — admins are expected to use the sentinel as the only entry, but ANY occurrence flips the channel. Exact-match (not substring) avoids accidental matches inside long real keys.

**Alternatives considered:**
- Only when ALL keys are sentinel — rejected: silent fallback to "real keys + BYOK" is confusing and dangerous.
- Only when single-key mode AND key equals sentinel — rejected: needlessly restrictive.

### Decision 6: Channel test with admin-supplied upstream key

**Choice:** Extend the test request DTO with an optional `byok_test_key` string field. In `controller/channel-test.go`, after `info.InitChannelMeta`, if the selected channel is BYOK:

- If `byok_test_key` is non-empty → override `info.ApiKey` with it for the duration of the test.
- If empty → return a test error with message `"BYOK channel requires test upstream key"`.

**Rationale:** A test without an upstream key would always fail at the provider and silently disable a working BYOK channel via auto-ban. Requiring the admin to supply a one-shot test key keeps the test meaningful without persisting any credential.

### Decision 7: UI hint and copy affordance

**Choice:** Both themes show a helper line under the Key textarea with the copy-able `$FORWARD_KEY` sentinel and a one-line explanation. The channel test dialog shows an extra "Upstream test key (BYOK)" input only when the current channel's stored Key contains the sentinel (detected client-side by the same exact-line match the backend uses). New i18n keys are added across en/zh/fr/ru/ja/vi for both themes.

**Rationale:** The sentinel is operator-visible state, so making it discoverable in-form (rather than buried in docs) lowers the chance of typos. Hiding the BYOK test input behind a runtime check keeps the test dialog uncluttered for normal channels.

## Risks / Trade-offs

- **Composite-key channels (AWS, Vertex JSON, Xunfei, Volcengine TTS) work transparently but require the client to send the full composite string after `:` (e.g., `sk-token:access|secret|region`).** → Mitigation: UI hint explicitly states "BYOK works best with simple-key providers" and the docs note the composite-key format expectation.
- **A real provider key that happens to start with the sentinel substring (e.g., `$FORWARD_KEY_x`) would NOT flip the channel** (we use exact-line match), but operators might be confused. → Mitigation: hint clarifies the sentinel must be the entire key value.
- **URL prefix `/sk-...` could shadow a future intentional route starting with `/sk-`.** → Mitigation: NoRoute only fires when no real route matched, so any explicit route registration takes precedence.
- **Channel auto-ban could disable a BYOK channel after a few client-side wrong-key requests (HTTP 401s from upstream).** → Mitigation: not addressed in this change; documented as a known trade-off. Admins can disable auto-ban per channel as they do today, or a future change can add a `bypass_auto_ban_on_byok` flag.
- **WebSocket realtime path (`/v1/realtime`) carries the new-api key via `Sec-WebSocket-Protocol: openai-insecure-api-key.<key>`** (`auth.go:282-291`). The `:` separator must work inside that subprotocol segment too. → Mitigation: TokenAuth applies the `:` split to the post-`openai-insecure-api-key.` value before existing parsing.
- **`x-api-key` for Claude paths and `x-goog-api-key`/`?key=` for Gemini are also rewritten into `Authorization: Bearer sk-...` by existing TokenAuth before parsing.** → No new mitigation required: the `:` split is applied at the same single point, so all carriers behave identically.

## Migration Plan

No runtime migration. Existing channels keep working without any modification. Operators opt into BYOK by editing a channel and replacing the Key value with `$FORWARD_KEY`. Rollback is symmetric: replace `$FORWARD_KEY` with a real key to restore stored-key mode.

## Open Questions

None — all eight discovery questions from the conversation were locked before spec creation.
