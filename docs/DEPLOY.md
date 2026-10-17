# Deploy — Oracle host, behind Caddy

Target: the Always Free Micro that runs `v2v-engine` and `v2v-bot` (`ssh v2vdemo`).
Public surface: **80/443 → Caddy → 127.0.0.1:8085 (this server) → 127.0.0.1:8091
(engine web API)**. Everything is operated with `minions/srv.sh` in v2v-demo
(`srv.sh push-bin tma`, `status`, `logs tma`, `rollback tma`).

## 0. Prerequisites
- DNS **A record** `app.pere.org.ua → <server public IP>` (Cloudflare zone, DNS only).
  Check: `dig +short app.pere.org.ua`.
- Oracle VCN Security List: **ingress** TCP 80 and 443 from 0.0.0.0/0.
- Host firewall (iptables, before the final `REJECT`), then persist:
  ```
  sudo iptables -I INPUT 5 -p tcp --dport 80  -m state --state NEW -j ACCEPT
  sudo iptables -I INPUT 5 -p tcp --dport 443 -m state --state NEW -j ACCEPT
  sudo netfilter-persistent save
  ```

## 1. Engine (server `~/v2v-demo/.env`)
```
WEB_ADDR=127.0.0.1:8091
TMA_BOT_TOKEN=<the token of the bot that hosts the Mini App = v2v-bot's TELEGRAM_BOT_TOKEN>
# CHECKOUT=stars   # only after HTTPS works and the bot relays payments (docs/HANDOFF.md)
```
Restart (`srv.sh restart engine`); the journal must say `web API on http://127.0.0.1:8091`.

## 2. This server: unit, env, binary
`~/v2v-tma/.env` (chmod 600, no quotes):
```
ENGINE_URL=http://127.0.0.1:8091
ADDR=127.0.0.1:8085
```
Install `deploy/v2v-tma.service` to `/etc/systemd/system/`, `sudo systemctl daemon-reload &&
sudo systemctl enable v2v-tma`, then `srv.sh push-bin tma` (cross-compiles, uploads, restarts).
The client is embedded in the binary.

## 3. Caddy
`sudo apt install caddy`, put `deploy/Caddyfile` in `/etc/caddy/Caddyfile`,
`sudo systemctl reload caddy`. The certificate is issued on the first request.

## 4. Verify, then register
```
curl -sI https://app.pere.org.ua/             # 200
curl -s  https://app.pere.org.ua/v1/topics    # 10 topics
```
Then @BotFather → the bot → Menu Button / Mini App URL `https://app.pere.org.ua`.
Inside Telegram the requests carry `initData`; outside it they get `401` (correct).

## Rollback
`srv.sh rollback tma`, or `sudo systemctl disable --now v2v-tma caddy` and drop the two
iptables rules (`-D`). The engine's `WEB_ADDR` can stay: it is loopback.
