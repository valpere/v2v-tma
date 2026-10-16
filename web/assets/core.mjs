// Pure helpers of the Mini App: no DOM, no Telegram SDK, so `node --test` covers them.

export function esc(s) {
  return String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

export function qtyOf(cart, itemId) {
  return cart.items.find((c) => c.item_id === itemId)?.quantity || 0;
}

// Ukrainian plural: 1 / 2-4 / 0, 5-20 (21 is "one" again, 22 "few").
export function plural(n, one, few, many) {
  const m10 = n % 10;
  const m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return one;
  if (m10 >= 2 && m10 <= 4 && (m100 < 12 || m100 > 14)) return few;
  return many;
}

// The engine detects the language itself; this is only the hint the contract carries.
export function langOf(code) {
  return String(code || '').toLowerCase().startsWith('en') ? 'en' : 'uk';
}

export function secondsOf(startMs, nowMs) {
  return Math.max(1, Math.round((nowMs - startMs) / 1000));
}

export function b64ToBytes(b64) {
  return Uint8Array.from(atob(b64), (c) => c.charCodeAt(0));
}

// makeApi binds fetch and the Telegram initData: every call carries the credentials,
// the server enforces them.
export function makeApi(fetchFn, initData) {
  const auth = initData ? { Authorization: 'tma ' + initData } : {};
  async function run(url, init) {
    const res = await fetchFn(url, init);
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error || 'HTTP ' + res.status);
    return data;
  }
  return {
    json: (method, path, body) =>
      run(path, { method, headers: { 'Content-Type': 'application/json', ...auth }, body: body ? JSON.stringify(body) : undefined }),
    // the recording goes up raw; its container is the Content-Type
    voice: (sessionId, blob, seconds) =>
      run(`/v1/sessions/${sessionId}/voice?seconds=${seconds}`, { method: 'POST', headers: { 'Content-Type': blob.type, ...auth }, body: blob }),
  };
}
