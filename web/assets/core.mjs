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
    // speak=false (sound off): the engine skips the speech synthesis, which is the part that costs money
    voice: (sessionId, blob, seconds, speak = true) =>
      run(`/v1/sessions/${sessionId}/voice?seconds=${seconds}${speak ? '' : '&speak=0'}`, { method: 'POST', headers: { 'Content-Type': blob.type, ...auth }, body: blob }),
  };
}

// encodeWav turns captured mono float chunks (Web Audio, any sample rate) into a 16 kHz PCM16 WAV:
// the recorder's fallback where MediaRecorder is missing (Telegram Desktop's webview). 16 kHz is
// what speech-to-text wants anyway and keeps 30 s under the engine's 1 MiB upload cap.
export function encodeWav(chunks, inRate, outRate = 16000) {
  const n = chunks.reduce((s, c) => s + c.length, 0);
  const all = new Float32Array(n);
  let o = 0;
  for (const c of chunks) { all.set(c, o); o += c.length; }
  const ratio = inRate / outRate;
  const m = Math.floor(n / ratio);
  const out = new Uint8Array(44 + m * 2);
  const v = new DataView(out.buffer);
  const str = (off, s) => { for (let i = 0; i < s.length; i++) v.setUint8(off + i, s.charCodeAt(i)); };
  str(0, 'RIFF'); v.setUint32(4, 36 + m * 2, true); str(8, 'WAVE'); str(12, 'fmt ');
  v.setUint32(16, 16, true); v.setUint16(20, 1, true); v.setUint16(22, 1, true);
  v.setUint32(24, outRate, true); v.setUint32(28, outRate * 2, true); v.setUint16(32, 2, true); v.setUint16(34, 16, true);
  str(36, 'data'); v.setUint32(40, m * 2, true);
  for (let i = 0; i < m; i++) {
    const a = Math.floor(i * ratio);
    const b = Math.min(n, Math.max(a + 1, Math.floor((i + 1) * ratio)));
    let sum = 0;
    for (let j = a; j < b; j++) sum += all[j];
    const s = Math.max(-1, Math.min(1, sum / (b - a)));
    v.setInt16(44 + i * 2, s < 0 ? s * 32768 : s * 32767, true);
  }
  return out;
}
