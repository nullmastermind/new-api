## ADDED Requirements

### Requirement: Sentinel-driven channel BYOK mode

The system SHALL treat the literal string `$FORWARD_KEY` as a sentinel value in the `channel.Key` column that flips the channel into Bring-Your-Own-Key (BYOK) forwarding mode. When BYOK mode is active, the system SHALL use a client-supplied upstream key in place of any stored channel key. The system SHALL detect BYOK mode by exact-line match against the sentinel, treating multi-key channels (newline-joined keys) as BYOK if **any** line equals the sentinel after trimming whitespace.

#### Scenario: Single-key channel with sentinel value

- **WHEN** an admin saves a channel whose `Key` field contains exactly `$FORWARD_KEY`
- **THEN** the channel is considered BYOK-enabled and any request routed to it SHALL require a client-supplied upstream key

#### Scenario: Multi-key channel with sentinel mixed in

- **WHEN** a multi-key channel stores keys `realkey1\n$FORWARD_KEY\nrealkey3`
- **THEN** the channel is considered BYOK-enabled and the upstream key from the client request takes precedence over any stored key

#### Scenario: Real key that contains the sentinel as substring

- **WHEN** a channel's `Key` line is `$FORWARD_KEY_extra` (sentinel as prefix only)
- **THEN** the channel SHALL NOT be considered BYOK-enabled (exact-line match, not substring)

### Requirement: Header-based upstream-key carrier format

The system SHALL accept the client carrier format `sk-<new-api-token>:<upstream-key>` in every header source that `TokenAuth` reads today: `Authorization` (with `Bearer ` or `bearer ` prefix), `x-api-key`, `x-goog-api-key`, `mj-api-secret`, and the `openai-insecure-api-key.<value>` segment of the `Sec-WebSocket-Protocol` header. The system SHALL split the carrier value on the first `:` only (preserving any further `:` characters in the upstream half) and SHALL validate the left half as a new-api token through the existing parsing path unchanged.

#### Scenario: Authorization header carries token and upstream key

- **WHEN** a client sends `Authorization: Bearer sk-abc123:sk-real-openai-key`
- **THEN** the system SHALL validate `abc123` as the new-api token and SHALL store `sk-real-openai-key` as the upstream key in the request context

#### Scenario: Upstream key itself contains a colon

- **WHEN** a client sends `Authorization: Bearer sk-abc123:weird:upstream:value`
- **THEN** the system SHALL validate `abc123` as the new-api token and SHALL store `weird:upstream:value` as the upstream key (split occurs on first `:` only)

#### Scenario: No colon present (legacy format)

- **WHEN** a client sends `Authorization: Bearer sk-abc123-7-mygroup`
- **THEN** the system SHALL behave exactly as today — validate the token, parse channel ID `7` and group `mygroup`, and SHALL NOT set any upstream-key context value

#### Scenario: x-api-key carries token and upstream key (Claude path)

- **WHEN** a client sends `x-api-key: sk-abc123:sk-anthropic-real-key` to `/v1/messages`
- **THEN** the system SHALL apply the same split semantics as the `Authorization` header

#### Scenario: x-goog-api-key carries token and upstream key (Gemini path)

- **WHEN** a client sends `x-goog-api-key: sk-abc123:gemini-real-key` to a `/v1beta/models/...` path
- **THEN** the system SHALL apply the same split semantics

#### Scenario: WebSocket subprotocol carries token and upstream key

- **WHEN** a client opens a WebSocket with `Sec-WebSocket-Protocol: realtime, openai-insecure-api-key.sk-abc123:sk-real-openai-key, openai-beta.realtime-v1`
- **THEN** the system SHALL extract the `sk-abc123:sk-real-openai-key` portion, split on first `:`, and proceed as with a normal Authorization header

#### Scenario: Gemini ?key= query parameter carries token and upstream key

- **WHEN** a client sends `GET /v1beta/models/gemini-pro?key=sk-abc123:real-gemini-key`
- **THEN** the system SHALL apply the same split semantics

### Requirement: URL-prefix token carrier

The system SHALL accept a URL-prefix form of the new-api token: any request path matching the regex `^/sk-[A-Za-z0-9_-]+/` SHALL be rewritten so that the `sk-<token>` segment becomes the new-api token and the existing `Authorization` header (or gemini `?key=` query) becomes the upstream key. The rewrite SHALL be implemented via `engine.NoRoute` + `engine.HandleContext` so that no existing route is shadowed.

#### Scenario: URL-prefix carrier with bearer upstream key

- **WHEN** a client sends `POST /sk-abc123/v1/chat/completions` with `Authorization: Bearer sk-real-openai-key`
- **THEN** the system SHALL rewrite the path to `/v1/chat/completions` and the Authorization header to `Bearer sk-abc123:sk-real-openai-key`, then re-dispatch through normal routing

#### Scenario: URL-prefix carrier with empty Authorization (non-BYOK channel)

- **WHEN** a client sends `POST /sk-abc123/v1/chat/completions` with no Authorization header, targeting a non-BYOK channel
- **THEN** the system SHALL rewrite the path to `/v1/chat/completions` and the Authorization header to `Bearer sk-abc123` (no `:` appended), and the request SHALL proceed normally

#### Scenario: URL-prefix carrier with Authorization that already contains a colon

- **WHEN** a client sends `POST /sk-abc123/v1/chat/completions` with `Authorization: Bearer sk-other-token:sk-real-upstream-key`
- **THEN** the URL token SHALL win — the system rewrites the Authorization header to `Bearer sk-abc123:sk-real-upstream-key`, discarding the `sk-other-token` left half

#### Scenario: URL-prefix carrier with gemini ?key= query

- **WHEN** a client sends `GET /sk-abc123/v1beta/models/gemini-pro?key=real-gemini-key`
- **THEN** the system SHALL rewrite to `/v1beta/models/gemini-pro` with `Authorization: Bearer sk-abc123:real-gemini-key`, removing the consumed `?key=` parameter

#### Scenario: Path that matches a real route is not rewritten

- **WHEN** a client sends `POST /v1/chat/completions` (no `/sk-` prefix)
- **THEN** the NoRoute handler SHALL NOT fire and the request proceeds through normal routing unchanged

#### Scenario: URL prefix without trailing slash does not match

- **WHEN** a client sends `POST /sk-abc123` (no trailing slash, no further path)
- **THEN** the NoRoute handler SHALL NOT rewrite the request

### Requirement: BYOK channel enforcement at distributor

The system SHALL, after channel selection in `SetupContextForSelectedChannel`, check whether the chosen channel is BYOK-enabled and:
- If yes and an upstream key is present in the request context, set the channel-key context value to the upstream key (so `info.ApiKey` and all adapter logic see the upstream key transparently);
- If yes and no upstream key is present, abort the request with HTTP 401 and a documented message identifying the expected client format;
- If no, proceed exactly as today using the stored channel key.

#### Scenario: BYOK channel receives request with upstream key

- **WHEN** a request targeting a BYOK channel reaches the distributor with `ContextKeyBYOKUpstreamKey = "sk-real-upstream-key"`
- **THEN** the system SHALL set the channel-key context value to `sk-real-upstream-key` and continue request processing

#### Scenario: BYOK channel receives request without upstream key

- **WHEN** a request targeting a BYOK channel reaches the distributor with no `ContextKeyBYOKUpstreamKey` set (or set to empty)
- **THEN** the system SHALL abort with HTTP 401 and a response body containing the message `"this channel requires BYOK format: sk-<token>:<upstream-key>"`

#### Scenario: Non-BYOK channel ignores upstream-key context

- **WHEN** a request targeting a non-BYOK channel reaches the distributor, even if `ContextKeyBYOKUpstreamKey` happens to be set
- **THEN** the system SHALL use the stored channel key exactly as today and SHALL NOT consume the upstream-key context value

### Requirement: BYOK-aware channel test

The channel test endpoint SHALL accept an optional `byok_test_key` field on the test request DTO. When the channel under test is BYOK-enabled:
- If `byok_test_key` is non-empty, the test SHALL execute with that value as the upstream key;
- If `byok_test_key` is empty or absent, the test SHALL fail fast with a clear error rather than executing against the upstream with no real credential.

Non-BYOK channels SHALL ignore `byok_test_key` entirely.

#### Scenario: BYOK channel test with admin-supplied upstream key

- **WHEN** an admin runs a test on a BYOK channel with `byok_test_key = "sk-real-test-key"`
- **THEN** the test SHALL execute against the upstream using that key and report the upstream's actual response

#### Scenario: BYOK channel test without upstream key

- **WHEN** an admin runs a test on a BYOK channel with `byok_test_key` empty or absent
- **THEN** the test SHALL fail with a clear error message (`"BYOK channel requires test upstream key"` or equivalent) before any upstream call is made, and the channel SHALL NOT be auto-banned by the test failure

#### Scenario: Non-BYOK channel test ignores byok_test_key

- **WHEN** an admin runs a test on a non-BYOK channel and includes `byok_test_key`
- **THEN** the test SHALL run exactly as today, using the stored channel key, and the `byok_test_key` value SHALL be ignored

### Requirement: UI affordance for BYOK on channel form

The channel create/edit form in both `web/default` and `web/classic` themes SHALL show a helper line under the API Key field with:
- The literal sentinel `$FORWARD_KEY` rendered in a copy-able element (button, badge, or icon that copies to clipboard);
- A short explanation that pasting the sentinel as the key value enables BYOK forwarding;
- A note that BYOK works best with simple-key providers (OpenAI-compatible, Anthropic, Gemini, Azure) and that composite-key providers (AWS, Vertex JSON, Xunfei, Volcengine TTS) require the client to send the full composite string after `:`.

The channel test dialog in both themes SHALL show an additional "Upstream test key (BYOK)" text input only when the currently selected channel's stored Key contains the sentinel (matched line-by-line, exact). The dialog SHALL submit this value as `byok_test_key` to the test endpoint.

#### Scenario: Operator sees BYOK hint on channel form

- **WHEN** an operator opens the channel create or edit form
- **THEN** under the API Key field, a helper line SHALL display the copy-able `$FORWARD_KEY` sentinel and the BYOK explanation

#### Scenario: Operator copies the sentinel from the hint

- **WHEN** an operator clicks the copy affordance next to `$FORWARD_KEY` in the hint
- **THEN** the sentinel string SHALL be copied to the clipboard

#### Scenario: BYOK channel shows test-key input in test dialog

- **WHEN** an operator opens the test dialog for a channel whose stored Key contains the sentinel
- **THEN** the dialog SHALL display an "Upstream test key (BYOK)" input field

#### Scenario: Non-BYOK channel hides test-key input

- **WHEN** an operator opens the test dialog for a channel whose stored Key does not contain the sentinel
- **THEN** the dialog SHALL NOT display the "Upstream test key (BYOK)" input

#### Scenario: i18n keys exist in all supported locales

- **WHEN** the application renders the BYOK hint or test-key input in any of the supported languages (en, zh, fr, ru, ja, vi)
- **THEN** all user-facing strings SHALL have a translation defined in that locale's resource file for the active theme

### Requirement: BYOK does not alter quota, billing, logging, or adapter behavior

The system SHALL preserve all existing quota deduction, billing settlement, log attribution, retry, auto-ban, and per-adapter request-construction behavior for BYOK requests. The only difference between a BYOK and a non-BYOK request SHALL be the source of `info.ApiKey` (request context vs. database). The upstream key SHALL NOT be persisted, logged in cleartext, or otherwise exposed beyond the in-memory request lifecycle. `RelayInfo.String` masking of `ApiKey` SHALL continue to apply unchanged.

#### Scenario: Quota deducted from new-api token, not upstream

- **WHEN** a BYOK request completes successfully
- **THEN** quota SHALL be deducted from the user/token identified by the new-api token half of the carrier, exactly as today

#### Scenario: Log records new-api token attribution

- **WHEN** a BYOK request is logged
- **THEN** the log entry SHALL reference the new-api token (id, name, group) and SHALL NOT contain the upstream key in cleartext

#### Scenario: Adapter receives upstream key in info.ApiKey

- **WHEN** an adapter's `SetupRequestHeader` is invoked for a BYOK request
- **THEN** `info.ApiKey` SHALL contain the client-supplied upstream key and the adapter SHALL set the provider-specific header (e.g. `Authorization: Bearer`, `x-api-key`, `x-goog-api-key`, `api-key`) using that value with no adapter-level code change required

#### Scenario: RelayInfo string representation masks ApiKey

- **WHEN** a BYOK `RelayInfo` is converted to string via `RelayInfo.String()`
- **THEN** the output SHALL show `ApiKey: ***masked***` regardless of whether the key originated from the database or the request context
