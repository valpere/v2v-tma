import test from 'node:test';
import assert from 'node:assert/strict';
import { esc, qtyOf, plural, langOf, secondsOf, makeApi, b64ToBytes } from '../assets/core.mjs';

test('esc escapes markup and tolerates null', () => {
  assert.equal(esc('<b a="x">&\''), '&lt;b a=&quot;x&quot;&gt;&amp;&#39;');
  assert.equal(esc(null), '');
  assert.equal(esc(5), '5');
});

test('qtyOf reads the server cart', () => {
  const cart = { items: [{ item_id: 'a', quantity: 2 }] };
  assert.equal(qtyOf(cart, 'a'), 2);
  assert.equal(qtyOf(cart, 'b'), 0);
  assert.equal(qtyOf({ items: [] }, 'a'), 0);
});

test('plural follows Ukrainian rules', () => {
  const f = (n) => plural(n, 'позиція', 'позиції', 'позицій');
  assert.deepEqual([1, 2, 4, 5, 11, 12, 21, 22, 25, 101].map(f),
    ['позиція', 'позиції', 'позиції', 'позицій', 'позицій', 'позицій', 'позиція', 'позиції', 'позицій', 'позиція']);
});

test('langOf maps the Telegram language to a supported one', () => {
  assert.equal(langOf('en'), 'en');
  assert.equal(langOf('en-GB'), 'en');
  assert.equal(langOf('uk'), 'uk');
  assert.equal(langOf('ru'), 'uk');
  assert.equal(langOf(undefined), 'uk');
});

test('secondsOf is at least one, rounded', () => {
  assert.equal(secondsOf(1000, 1100), 1);
  assert.equal(secondsOf(0, 2600), 3);
});

test('b64ToBytes decodes', () => {
  assert.deepEqual([...b64ToBytes(Buffer.from('OggS').toString('base64'))], [79, 103, 103, 83]);
});

function fakeFetch(status, body) {
  const calls = [];
  const f = async (url, init) => { calls.push({ url, init }); return { ok: status < 400, status, json: async () => body }; };
  f.calls = calls;
  return f;
}

test('api sends initData and JSON, returns the body', async () => {
  const f = fakeFetch(200, { ok: 1 });
  const api = makeApi(f, 'INIT');
  assert.deepEqual(await api.json('POST', '/v1/x', { a: 1 }), { ok: 1 });
  const { init } = f.calls[0];
  assert.equal(init.headers.Authorization, 'tma INIT');
  assert.equal(init.headers['Content-Type'], 'application/json');
  assert.equal(init.body, '{"a":1}');
});

test('api omits Authorization without initData and surfaces the server error', async () => {
  const f = fakeFetch(429, { error: 'ліміт' });
  const api = makeApi(f, '');
  await assert.rejects(api.json('GET', '/v1/x'), /ліміт/);
  assert.equal(f.calls[0].init.headers.Authorization, undefined);
});

test('api falls back to the HTTP status when the error body is not JSON', async () => {
  const f = async () => ({ ok: false, status: 502, json: async () => { throw new Error('x'); } });
  await assert.rejects(makeApi(f, '').json('GET', '/v1/x'), /HTTP 502/);
});

test('voice posts the raw recording with its own content type and duration', async () => {
  const f = fakeFetch(200, { transcript: 'hi' });
  const blob = { type: 'audio/webm;codecs=opus', size: 3 };
  assert.deepEqual(await makeApi(f, 'INIT').voice('sid', blob, 4), { transcript: 'hi' });
  const { url, init } = f.calls[0];
  assert.equal(url, '/v1/sessions/sid/voice?seconds=4');
  assert.equal(init.headers['Content-Type'], 'audio/webm;codecs=opus');
  assert.equal(init.headers.Authorization, 'tma INIT');
  assert.equal(init.body, blob);
});

import { encodeWav } from '../assets/core.mjs';

const u32 = (b, o) => new DataView(b.buffer, b.byteOffset).getUint32(o, true);
const u16 = (b, o) => new DataView(b.buffer, b.byteOffset).getUint16(o, true);
const i16 = (b, o) => new DataView(b.buffer, b.byteOffset).getInt16(o, true);
const tag = (b, o) => String.fromCharCode(...b.slice(o, o + 4));

test('encodeWav writes a 16 kHz mono PCM16 file, downsampling by averaging', () => {
  const x = new Float32Array(48000).fill(0.5); // one second at 48 kHz
  const wav = encodeWav([x.subarray(0, 20000), x.subarray(20000)], 48000);
  assert.equal(tag(wav, 0), 'RIFF');
  assert.equal(tag(wav, 8), 'WAVE');
  assert.equal(u16(wav, 20), 1); // PCM
  assert.equal(u16(wav, 22), 1); // mono
  assert.equal(u32(wav, 24), 16000);
  assert.equal(u16(wav, 34), 16);
  assert.equal(u32(wav, 40), 32000); // 16000 samples * 2 bytes
  assert.equal(wav.length, 44 + 32000);
  assert.equal(u32(wav, 4), wav.length - 8);
  assert.equal(i16(wav, 44), 16383); // 0.5 * 32767, truncated
});

test('encodeWav clamps and survives silence and empty input', () => {
  const loud = encodeWav([new Float32Array([2, -2, 2, -2])], 16000);
  assert.equal(i16(loud, 44), 32767);
  assert.equal(i16(loud, 46), -32768);
  assert.equal(encodeWav([], 16000).length, 44);
});

test('voice URL asks the engine to skip speech when sound is off', async () => {
  const f = fakeFetch(200, {});
  const blob = { type: 'audio/wav', size: 1 };
  await makeApi(f, '').voice('s', blob, 2, false);
  await makeApi(f, '').voice('s', blob, 2, true);
  assert.equal(f.calls[0].url, '/v1/sessions/s/voice?seconds=2&speak=0');
  assert.equal(f.calls[1].url, '/v1/sessions/s/voice?seconds=2');
});

import { welcomeOf } from '../assets/core.mjs';

test('welcomeOf: the topic welcome, but never a repeat of the tagline shown above it', () => {
  assert.equal(welcomeOf({ welcome: 'Вітаю!', tagline: 'Піца' }), 'Вітаю!');
  assert.match(welcomeOf({ welcome: 'Піца', tagline: 'Піца' }), /Чим можу допомогти/); // the engine falls back to the tagline
  assert.match(welcomeOf({ tagline: 'Піца' }), /Чим можу допомогти/);
});
