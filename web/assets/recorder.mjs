// Microphone capture for the cloud-STT path: record until stop(), resolve a Blob and its length.
// MediaRecorder where it works; otherwise raw Web Audio samples encoded as WAV (Telegram Desktop's
// webview has getUserMedia but throws "MediaRecorder is unsupported on this platform").
import { secondsOf, encodeWav } from './core.mjs';

export const MAX_SECONDS = 60; // the engine's VOICE_MAX_SECONDS may be lower; it then says so
const MAX_WAV_SECONDS = 30; // 16 kHz PCM16 is 32 KB/s: 30 s stays under the 1 MiB upload cap

export function recorderSupported() {
  return !!(navigator.mediaDevices?.getUserMedia && (window.MediaRecorder || window.AudioContext || window.webkitAudioContext));
}

// start() resolves a handle once the microphone is open (rejects if the user refuses);
// handle.done resolves {blob, seconds} after handle.stop() or the time limit.
export async function start() {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  const release = () => stream.getTracks().forEach((t) => t.stop());
  try {
    return startMediaRecorder(stream, release);
  } catch {
    // no encoder on this platform: fall through to raw samples
  }
  try {
    return startWav(stream, release);
  } catch (err) {
    release(); // never leave the mic open
    throw err;
  }
}

function startMediaRecorder(stream, release) {
  const mr = new MediaRecorder(stream); // throws where unsupported
  const chunks = [];
  const t0 = Date.now();
  let timer;
  const done = new Promise((resolve, reject) => {
    mr.onerror = (e) => { clearTimeout(timer); release(); reject(e.error || new Error('recorder error')); };
    mr.ondataavailable = (e) => e.data.size && chunks.push(e.data);
    mr.onstop = () => {
      clearTimeout(timer);
      release();
      resolve({ blob: new Blob(chunks, { type: mr.mimeType || 'audio/webm' }), seconds: secondsOf(t0, Date.now()) });
    };
  });
  mr.start();
  timer = setTimeout(() => mr.state !== 'inactive' && mr.stop(), MAX_SECONDS * 1000);
  return { stop: () => mr.state !== 'inactive' && mr.stop(), done };
}

function startWav(stream, release) {
  const Ctx = window.AudioContext || window.webkitAudioContext;
  const ctx = new Ctx();
  const src = ctx.createMediaStreamSource(stream);
  const proc = ctx.createScriptProcessor(4096, 1, 1);
  const mute = ctx.createGain(); // the node must be connected to run; silence keeps it out of the speakers
  mute.gain.value = 0;
  const chunks = [];
  const t0 = Date.now();
  proc.onaudioprocess = (e) => chunks.push(new Float32Array(e.inputBuffer.getChannelData(0)));
  src.connect(proc); proc.connect(mute); mute.connect(ctx.destination);
  let finish;
  const done = new Promise((resolve) => {
    finish = () => {
      clearTimeout(timer);
      proc.onaudioprocess = null;
      src.disconnect(); proc.disconnect(); mute.disconnect();
      release();
      ctx.close();
      resolve({ blob: new Blob([encodeWav(chunks, ctx.sampleRate)], { type: 'audio/wav' }), seconds: secondsOf(t0, Date.now()) });
    };
  });
  let stopped = false;
  const stop = () => { if (!stopped) { stopped = true; finish(); } };
  const timer = setTimeout(stop, MAX_WAV_SECONDS * 1000);
  return { stop, done };
}
