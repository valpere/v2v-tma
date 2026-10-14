# AGENTS.md — v2v-tma

An interactive **Telegram Mini App (TMA)** featuring a dual-mode (Voice & Text) AI Concierge for multi-domain commerce with real-time **Voice-to-UI synchronization** and native **Telegram Stars (XTR)** payments.

Built as a verified, high-converting showcase and reusable architectural core for client bids and live demonstrations. Follow these directives strictly when working on or extending this codebase.

---

## 1. Core Philosophy & Architectural Guardrails

1. **Dual-Mode (Zero-Cost Default):**
   - **Text mode is the default.** Do not make speculative or background cloud STT/TTS API calls (ElevenLabs, Azure, Whisper) on simple catalog navigation or text queries.
   - For audio feedback in text mode, prioritize client-side native Web Speech API (`SpeechSynthesis`) at $0 cost.
   - Cloud Voice (STT/TTS) is **strictly On-Demand**, triggered only when the user explicitly enables voice or clicks Push-to-Talk.

2. **Voice-to-UI Synchronization Protocol (`ui_actions`):**
   - The LLM and backend MUST NOT return plain unstructured text. Every intent response MUST contain a structured array of `ui_actions` (e.g. `HIGHLIGHT_ITEM`, `ADD_TO_CART`, `OPEN_CHECKOUT`).
   - The UI MUST react visually to user speech (animating cards, updating the cart, scrolling, firing Haptic Feedback).

3. **Telegram Platform Native Standards:**
   - Always validate `Telegram.WebApp.initData` via HMAC-SHA256 (`auth/telegram.go`) before processing secure operations.
   - Currency for digital items in Telegram Mini Apps MUST be **`XTR` (Telegram Stars)** using official Bot API `createInvoiceLink` / `Telegram.WebApp.openInvoice`.
   - All cart/order calculations MUST be authoritative on the server side — the client sends only `item_id` and `quantity`.
   - Keep tactile Haptic Feedback (`Telegram.WebApp.HapticFeedback`) on all key user actions.

---

## 2. Code Conventions & Standards

- **Language & Runtime:** Go 1.22+. Standard library first (`net/http`). Avoid bloated frameworks.
- **Frontend Stack:** Pure HTML5, modern vanilla JavaScript (ES6+), TailwindCSS (dark mode), Telegram WebApp SDK. No heavy React/Angular bundle bloat — keep initial FCP under 1.2 seconds and bundle under 150 KB.
- **Testing:** `go test ./...` must always pass. Keep domain rules and intent parsers covered by unit tests in `internal/domain/engine_test.go`.
- **Environment & Secrets:**
  - Never commit actual API keys or bot tokens. Use environment variables:
    - `TELEGRAM_BOT_TOKEN`
    - `PORT` (default: `8085`)
- **Directory Layout:**
  - `cmd/server/` — HTTP entrypoint and route handlers.
  - `internal/domain/` — Multi-domain catalog, business rules, query processing, and UI action extraction.
  - `internal/auth/` — Telegram `initData` HMAC-SHA256 validation.
  - `internal/payments/` — Telegram Stars payments via Bot API.
  - `web/` — Static Mini App frontend (`index.html`, `assets/app.js`).
  - `docs/` — Formal specifications (`USECASES.md`, `REQUIREMENTS.md`, `ARCHITECTURE_HLD.md`).
  - `tmp/` — Scratch scripts and temporary files (gitignored).

---

## 3. Multi-Domain Catalog Reference

The core ships with 4 pre-configured domains:
- `pizzeria`: "Папа Пепе" (Neapolitan pizza, spicy recommendations, cart checkout).
- `barbershop`: "Барбароса" (Grooming, beard trims, royal combos).
- `auto`: "Подбай-Авто" (ECU diagnostics, oil service, brake inspections).
- `lyapko`: "Lyapko Shop" (Health applicators, needle mats, rollers).

When adding new domains, register them in `internal/domain/catalog.go` following the `DomainInfo` and `Item` structs.

---

## 4. Verification Workflow

Before concluding any task or pushing changes:
```bash
# 1. Format and vet code
go fmt ./...
go vet ./...

# 2. Run unit tests
go test -v ./...

# 3. Build executable
go build -o bin/server ./cmd/server

# 4. Verify local server starts cleanly
./bin/server &
PID=$!
sleep 1
curl -s http://localhost:8085/api/domains | grep -q "pizzeria"
kill $PID
```
