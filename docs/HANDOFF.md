# Handoff — what this repo is and what the next agent should do

## Status: live prototype, not an MVP
`make run` (needs `DEV_AUTH=1`, already in the target) serves the Mini App and a
**stub** of the `/v1` contract (`docs/API.md`) on :8085. Verified: `make check`
green; a headless browser renders the topics and catalog from `/v1/topics`; a
curl run of message → server cart → checkout works. **Not verified:** inside
real Telegram (needs HTTPS + a bot with a Mini App), real Stars payment, voice.

## Target architecture (decided 2026-10-14)
```
 Telegram bot ─┐
 this TMA     ─┼──►  engine API /v1  (v2v-demo, Go)  ──► LLM / STT / TTS
 plain web    ─┘      sessions, catalog, cart, limits
```
This repo is **a thin client**. The engine lives in `~/wrk/demo/v2v-demo/v2v-demo`
(plan: its `tmp/engine-api-plan.md`, not committed, repo is public). Today the
stub in `cmd/server` + `internal/{domain,session,auth,payments}` stands in for
it; with `ENGINE_URL=http://engine:port` the same server proxies `/v1/*` there
instead and the stub is not used.

## What is real vs stub
| Real (keep) | Stub (replaced by the engine) |
|---|---|
| `web/` client (Telegram SDK, haptics, Web Speech, server-owned cart UI) | `internal/domain/engine.go` keyword matcher — no LLM |
| initData validation (`internal/auth`, enforced, 24 h, tests) | `internal/session` in-memory store |
| Server-side pricing/checkout (`domain.Quote`, no client amounts) | `internal/domain/catalog/*.json` is a copy; target home is `topics/<id>/catalog.json` in v2v-demo |
| Invoice creation (`internal/payments`) | Payment confirmation does not exist |

## Backlog, in order
1. **Engine side first** (other repo): extract `internal/engine` from `cmd/bot`, add `/v1` per `docs/API.md`, catalog + deterministic cart, Stars checkout. Until then nothing below that depends on the LLM is possible.
2. Switch this repo to the engine: set `ENGINE_URL`, run both, fix contract drift (update `docs/API.md` first). Then delete `internal/{domain,session,payments}` and `cmd/server/api.go`; keep only static serving + the proxy (and drop `auth` if the engine validates).
3. **HTTPS**: Telegram only opens Mini Apps over HTTPS. Pick Caddy on the Oracle box or a tunnel; register the Mini App URL with @BotFather.
4. **Payment confirmation** (engine): answer `pre_checkout_query` within 10 s, handle `successful_payment`, mark the order paid idempotently by `order_id`; the client currently clears the cart on `openInvoice` status `paid` only.
5. **Voice**: client already does Web Speech (free). Cloud voice = `POST .../voice` + `GET /v1/tts` on the engine, on demand only.
6. **Frontend debt**: Tailwind Play CDN (runtime JIT, violates the 150 KB / 1.2 s targets) → precompiled CSS; quick prompts are Ukrainian only; `lang` is hard-coded `uk`; no i18n; `app.js` is one 400-line IIFE (split only when it hurts); use Telegram `themeParams` and `MainButton` as the docs claim.
7. **Tests for the client** (none today besides `node --check`).

## Rules for the next agent
- `make check` must pass before every commit; the repo is **public** — no tokens, no `.env*`, no description of an open weakness in commit subjects/issues before it is fixed.
- Never add a client-supplied price or amount to any endpoint. Never re-introduce an auth bypass keyed on request data; `DEV_AUTH=1` is the only dev switch and the server refuses to start without a token, `DEV_AUTH` or `ENGINE_URL`.
- Do not grow the stub: every feature added to `internal/domain/engine.go` is throwaway once the engine exists.
- Code and comments in English; user-facing strings may stay Ukrainian.
