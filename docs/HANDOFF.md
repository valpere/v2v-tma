# Handoff — v2v-tma

## Status (2026-10-15): thin client on the real engine
The stub engine is gone. `cmd/server` embeds the Mini App and forwards `/v1/*` to
the engine (`ENGINE_URL` = v2v-demo `WEB_ADDR`). Verified locally end to end through
this server against a real engine (`WEB_DEV_AUTH=1`, `CHECKOUT=mock`): 10 topics,
session, message with `ui_actions` + server cart, mock checkout. **Not verified:**
inside real Telegram (needs HTTPS + a bot with a Mini App URL), a real Stars
payment, voice.

```
 Telegram bot (v2v-bot) ─┐
 this TMA                ─┼──►  engine /v1 (v2v-demo)  ──► LLM / STT / TTS
 plain web (v2v-web)     ─┘     sessions, catalog, cart, orders, limits
```

## What lives where
| Here | In the engine (v2v-demo) |
|---|---|
| `web/` client (Telegram SDK, haptics, Web Speech, cart UI) | `initData` validation (`TMA_BOT_TOKEN`), sessions, dialogue, catalog, cart |
| static + `/v1` proxy | Stars invoice (`CHECKOUT=stars`), orders, payment confirmation |
| | per-user daily cap (`WEB_PRINCIPAL_DAILY_TURNS`) |

The bot that hosts the Mini App must be the one whose token is the engine's
`TMA_BOT_TOKEN`, and **v2v-bot** (same token) relays `pre_checkout_query` /
`successful_payment` to the engine's channel API. That is already implemented.

## Backlog, in order
1. **HTTPS + register the Mini App.** Telegram opens Mini Apps only over HTTPS: same
   Caddy as v2v-web (see its `docs/DEPLOY.md`), a name such as `app.<domain>`, then
   @BotFather → the bot's Mini App URL. Engine env on the server: `WEB_ADDR`,
   `TMA_BOT_TOKEN`, `CHECKOUT=stars`, and the live bot's token must equal it.
2. **First real Stars payment** (small amount), then refund to verify the whole loop.
3. **Voice**: client does Web Speech (free). Cloud voice = `POST .../voice` +
   `GET /v1/tts` on the engine (not built), on demand only.
4. **Frontend debt**: Tailwind Play CDN (runtime JIT; violates the size/time targets)
   → precompiled CSS; quick prompts Ukrainian only; `lang` hard-coded `uk`; no i18n;
   `app.js` is one 400-line IIFE; use `themeParams` / `MainButton`.
5. **Tests for the client** (none besides `node --check`).

## Rules for the next agent
- `make check` must pass before every commit; the repo is **public** — no tokens, no
  `.env*`, no description of an open weakness before it is fixed.
- Never add a client-supplied price or amount to any endpoint; checkout is `{}`.
- Topics without a catalog (most of them) have nothing to order: the UI shows
  "лише розмова" and no cart.
- Code and comments in English; user-facing strings may stay Ukrainian.
