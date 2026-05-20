## Why

Today, every channel in new-api must store an upstream API key in its database row. Operators that want clients to bring their own upstream key (BYOK) — multi-tenant SaaS, agent platforms, internal billing pass-through, or simple key-isolation per user — have no first-class way to do it. Workarounds such as Header Override with `{client_header:Authorization}` do not work, because the value seen by new-api in the `Authorization` header is the new-api token itself, not the client's upstream key.

This change introduces a zero-schema-change BYOK mode: a sentinel value in `channel.Key` ("`$FORWARD_KEY`") flips the channel into forward-only mode, and clients supply the real upstream key inline. new-api continues to authenticate, attribute, log, and bill against its own token — only the upstream credential changes per request.

## What Changes

- **Sentinel-driven BYOK mode**: When `channel.Key` equals (or any line of a multi-key channel equals) the literal string `$FORWARD_KEY`, the channel forwards a client-supplied upstream key instead of using the DB value.
- **Token format extension (header-based)**: The `Authorization` (Bearer/bearer), `x-api-key`, `x-goog-api-key`, `mj-api-secret`, and `Sec-WebSocket-Protocol`'s `openai-insecure-api-key` carriers accept `sk-<new-api-token>:<upstream-key>`. The new-api token half is parsed and validated as before; the upstream half is stored per-request in the gin context.
- **URL-based token carrier (new)**: Any request path matching `^/sk-[A-Za-z0-9_-]+/` is rewritten via `engine.NoRoute` + `engine.HandleContext`. The `sk-...` segment becomes the new-api token, and the existing `Authorization` (or gemini `?key=`) becomes the upstream key. The rewrite synthesizes `Authorization: Bearer sk-<token>:<upstream>` and re-dispatches into the normal routes.
- **Distributor enforcement**: After channel selection, if the channel is in BYOK mode and no upstream key is present in the context, the request is aborted with HTTP 401 and a message documenting the expected `sk-<token>:<upstream-key>` format. Otherwise `info.ApiKey` is populated with the upstream key and all adapters work unchanged.
- **Channel test support**: The channel test endpoint accepts an optional `byok_test_key` field. For BYOK channels this field is required; without it the test returns an error rather than failing against the upstream.
- **UI hint (both themes)**: The channel create/edit form shows a helper line under the Key field with the copy-able `$FORWARD_KEY` sentinel and a short explanation. The channel test dialog shows an "Upstream test key (BYOK)" input when the channel is in BYOK mode.
- Adapters, DB schema, migrations, multi-key/test/auto-ban/quota/billing/log flows are **unchanged**.

## Capabilities

### New Capabilities

- `channel-byok`: Channel-level "Bring Your Own Key" mode. Covers sentinel detection, two client carrier formats (header `:` split and URL `/sk-token/` prefix), upstream-key propagation through the gin context into `info.ApiKey`, BYOK-aware channel testing, and the user-facing UI hint contract.

### Modified Capabilities

<!-- No existing specs exist in this repository's openspec/specs/ tree. Auth, distributor, and adapter behaviors are documented inline in code comments and CLAUDE.md, not as openspec capabilities. -->

## Impact

- **Auth middleware** (`middleware/auth.go`): Extended `TokenAuth` to split the carrier value on the first `:` before existing `sk-` and `-` parsing, and to store the upstream half in a new context key. Applies to all auth sources (header, `x-api-key`, `x-goog-api-key`, `mj-api-secret`, WebSocket subprotocol, gemini `?key=`).
- **Distributor** (`middleware/distributor.go`): After `channel.GetNextEnabledKey`, BYOK channels swap the DB key for the context upstream key or abort 401.
- **Channel model** (`model/channel.go`): Adds `IsForwardKeyMode()` helper that scans each `\n`-separated key line for an exact sentinel match.
- **Constants** (`constant/context_key.go`, new `constant/channel.go` or extension): New `ContextKeyBYOKUpstreamKey` and `ChannelKeyForwardSentinel` constants.
- **Router engine** (`router/main.go` or `main.go` engine setup): New `NoRoute` handler implementing the `/sk-<token>/...` rewrite contract.
- **Channel test** (`controller/channel-test.go` + DTO): New `byok_test_key` request field.
- **Frontend default theme** (`web/default/`): Hint + copy button on the Key field; BYOK test-key input in the test dialog; new i18n keys across en/zh/fr/ru/ja/vi.
- **Frontend classic theme** (`web/classic/`): Equivalent hint and test dialog changes; equivalent i18n keys.
- **Tests**: New unit tests for token parsing, sentinel detection, BYOK abort, and URL rewrite. Existing TokenAuth tests must continue to pass unchanged.
- **No database, no migration, no API-shape break**. The existing `channel.Key NOT NULL` validation is preserved (the sentinel string satisfies it).
- **Documented limitation**: Channels whose key is composite (AWS `access|secret|region`, Vertex JSON ADC, Xunfei `appid|secret|key`, Volcengine `appid|token`) work transparently but are not recommended for BYOK; the UI hint will state that BYOK works best with simple-key providers.
