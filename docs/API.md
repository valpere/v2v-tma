# /v1 API contract

The contract between any frontend (this Mini App, a plain web client, the bot)
and the engine. **v2v-demo is the intended implementation** (see
`docs/HANDOFF.md`); `cmd/server` ships a deterministic *stub* of the same
contract so the prototype runs standalone. Change this file first, then both
sides.

Auth: `Authorization: tma <Telegram.WebApp.initData>` on everything except
`GET /v1/topics`. Invalid, forged or older-than-24h initData is `401`. A missing
header is accepted only when the server runs with `DEV_AUTH=1`.
Errors are `{"error": "<message>"}` with a 4xx/5xx status.

| Method & path | Body | Response |
|---|---|---|
| `GET /v1/topics` | – | `[{id, order, name, tagline, icon, welcome, voice_sample, quick_prompts[], items[]}]` |
| `POST /v1/sessions` | `{topic}` | `201 {session_id, topic, cart}` — one session per topic, cart isolated per session |
| `POST /v1/sessions/{id}/messages` | `{text, lang}` (text 1..500 chars, lang `uk`\|`en`) | `{reply, signal, language, intent, ui_actions[], cart}` |
| `GET /v1/sessions/{id}/cart` | – | `cart` |
| `PUT /v1/sessions/{id}/cart/items/{item_id}` | `{quantity}` (0 removes, max 20) | `cart` |
| `DELETE /v1/sessions/{id}/cart` | – | `cart` |
| `POST /v1/sessions/{id}/checkout` | `{}` — **no amounts** | `{order_id, invoice_url, stars, total_uah, mock}` |

A session belongs to the principal that created it; any other caller gets `404`.

## Objects

- `item`: `{item_id, title, description, price_uah, price_stars, category, icon, badge?, keywords?}`
- `cart`: `{items:[{item_id,title,icon,price_uah,price_stars,quantity}], count, total_uah, total_stars}` — always computed by the server from the catalog; the client never states a price.
- `ui_action`: `{type, payload}` with `type` ∈ `HIGHLIGHT_ITEM` (`item_id`, `duration_ms`), `ADD_TO_CART` (`item_id`, `quantity` — informational: **the server has already applied it**, the response `cart` is the truth), `OPEN_CHECKOUT`.
- `signal`: `continue` \| `escalate` (the engine may also send `lead_ready`; clients must tolerate unknown values).

## Not in the contract yet (engine work)

Voice (`POST .../voice`, `GET /v1/tts`), payment confirmation
(`pre_checkout_query` / `successful_payment` → order marked paid), SSE streaming.
