# AGENTS.md — v2v-tma

Telegram Mini App (TMA): the **frontend** of the v2v-demo voice/text concierge,
with Voice-to-UI synchronisation and Telegram Stars (XTR) payments. Read
`docs/HANDOFF.md` first (status, ordered backlog). The API between this client and
the engine is `docs/API.md`.

## Architecture in one paragraph
This repo is a thin client: a small Go server embeds the Mini App (`web/`) and
forwards `/v1/*` to the engine (`~/wrk/demo/v2v-demo/v2v-demo`, `WEB_ADDR`). The
engine owns sessions, the grounded dialogue, catalog, cart, order amounts, Stars
invoices and payment confirmation; it validates the Telegram `initData` itself
(`TMA_BOT_TOKEN`), and the bot that hosts the Mini App (v2v-bot) relays the
payment updates to it. This server holds **no secrets and no business logic**.
Other frontends of the same engine: v2v-bot (Telegram), v2v-web (plain web).

## Guardrails
1. **Text mode is the default and costs nothing.** No speculative cloud STT/TTS;
   free client-side Web Speech (`SpeechSynthesis`) for audio in text mode; cloud
   voice only on explicit user action.
2. **Voice-to-UI:** every concierge turn returns `ui_actions` (`HIGHLIGHT_ITEM`,
   `ADD_TO_CART`, `OPEN_CHECKOUT`); the UI reacts (animation, haptics). Never
   plain unstructured text.
3. **Money is server-side only.** Clients send item ids and quantities, never a
   price or an amount; the checkout request carries no amounts at all (`{}`).
   Currency is `XTR`; the engine creates the invoice, the client only calls
   `Telegram.WebApp.openInvoice(invoice_url)`.
4. **Auth is the engine's.** `Authorization: tma <initData>` is forwarded
   untouched; there is no bypass here. Local development uses the engine's
   `WEB_DEV_AUTH=1` (loopback only). There are no cookies, so there is no CSRF
   surface: the credential is an explicit header.
5. Haptic feedback on all key user actions. Model output is rendered with
   `textContent`, never `innerHTML`; catalog fields go through `esc()`.

## Conventions
- Go 1.26+, standard library only, no frameworks. Frontend: vanilla ES6+, Tailwind
  (CDN — see HANDOFF debt), Telegram WebApp SDK; no bundler.
- Code and comments in English; user-facing strings may be Ukrainian.
- `make check` (gofmt, vet, `go test -race`, `node --check`, build) must pass
  before every commit. Test first for bug fixes.
- No secrets in code or logs; env only: `ENGINE_URL` (required), `ADDR`
  (127.0.0.1:8085), `WEB_DIR` (empty = embedded client). Never commit `.env*`. The
  repo is **public**: do not describe open weaknesses in commits/issues before
  they are fixed.
- `tmp/` is gitignored scratch; reusable helpers go to `minions/` (Bash, not Python).
- Do not write to other repositories without explicit approval from Valentyn.

## Layout
- `cmd/server/` — wiring, graceful shutdown · `internal/server/` — static + `/v1` proxy
- `web.go` — embeds `web/` (the Mini App) · `docs/` — `API.md`, `HANDOFF.md`, HLD, requirements, use cases

## Verification
`make check`; run the engine with `WEB_ADDR=127.0.0.1:8091 WEB_DEV_AUTH=1 CHECKOUT=mock`,
then `make run` and `curl localhost:8085/v1/topics` (10 topics; the ones with a
catalog: auto, barbershop, pizzeria, lyapko).
