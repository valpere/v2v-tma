# AGENTS.md — v2v-tma

Telegram Mini App (TMA): the **frontend** of the v2v-demo voice/text concierge,
with Voice-to-UI synchronisation and Telegram Stars (XTR) payments. **Live
prototype, not an MVP** — read `docs/HANDOFF.md` first (status, what is stub,
ordered backlog). The API between this client and the engine is `docs/API.md`.

## Architecture in one paragraph
This repo is a thin client. The engine (sessions, catalog, cart, LLM, STT/TTS,
payment confirmation) belongs to `~/wrk/demo/v2v-demo/v2v-demo` and will be
reached over `/v1` (`ENGINE_URL` makes `cmd/server` proxy to it). Until the
engine exposes `/v1`, `cmd/server` answers it with a deterministic **stub**. Other
frontends (plain web, the bot) are clients of the same API.

## Guardrails
1. **Text mode is the default and costs nothing.** No speculative cloud STT/TTS;
   free client-side Web Speech (`SpeechSynthesis`) for audio in text mode; cloud
   voice only on explicit user action.
2. **Voice-to-UI:** every concierge turn returns `ui_actions` (`HIGHLIGHT_ITEM`,
   `ADD_TO_CART`, `OPEN_CHECKOUT`); the UI reacts (animation, haptics). Never
   plain unstructured text.
3. **Money is server-side only.** Clients send item ids and quantities, never a
   price or an amount; the checkout request carries no amounts at all. Currency
   is `XTR` via `createInvoiceLink` + `Telegram.WebApp.openInvoice`.
4. **Auth is enforced.** `Authorization: tma <initData>` is validated (HMAC,
   24 h). No bypass keyed on request data; `DEV_AUTH=1` is the only dev switch.
5. Haptic feedback on all key user actions.

## Conventions
- Go 1.26+, standard library first (`net/http` mux with method patterns); no
  frameworks. Frontend: vanilla ES6+, Tailwind, Telegram WebApp SDK; no bundler.
- Code and comments in English; user-facing strings may be Ukrainian.
- `make check` (gofmt, vet, `go test -race`, `node --check`, build) must pass
  before every commit. Test first for bug fixes.
- No secrets in code or logs; env only: `TELEGRAM_BOT_TOKEN`, `PORT` (8085),
  `DEV_AUTH`, `ENGINE_URL`, `WEB_DIR`. Never commit `.env*`. The repo is **public**:
  do not describe open weaknesses in commits/issues before they are fixed.
- `tmp/` is gitignored scratch; reusable helpers go to `minions/` (Bash, not Python).
- Do not grow the stub (`internal/domain/engine.go`): it is throwaway once the
  engine exists.

## Layout
- `cmd/server/` — static files, `/v1` stub handlers (`api.go`), `ENGINE_URL` proxy
- `internal/domain/` — catalog (`catalog/*.json`, embedded), `Quote` pricing, stub matcher
- `internal/session/` — in-memory sessions bound to the principal
- `internal/auth/` — initData validation · `internal/payments/` — Stars invoices
- `web/` — the Mini App · `docs/` — `API.md`, `HANDOFF.md`, HLD, requirements, use cases

## Verification
`make check`, then `DEV_AUTH=1 PORT=8095 ./bin/server` and
`curl localhost:8095/v1/topics` lists 4 topics.
