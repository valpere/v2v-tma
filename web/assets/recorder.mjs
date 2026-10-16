// Microphone capture for the cloud-STT path: record until stop(), resolve a Blob and its length.
import { secondsOf } from './core.mjs';

export const MAX_SECONDS = 60; // the engine's VOICE_MAX_SECONDS may be lower; it then says so

export function recorderSupported() {
  return !!(navigator.mediaDevices?.getUserMedia && window.MediaRecorder);
}

// start() resolves a handle once the microphone is open (rejects if the user refuses);
// handle.done resolves {blob, seconds} after handle.stop() or MAX_SECONDS.
export async function start() {
  const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  const mr = new MediaRecorder(stream);
  const chunks = [];
  const t0 = Date.now();
  let timer;
  const done = new Promise((resolve) => {
    mr.ondataavailable = (e) => e.data.size && chunks.push(e.data);
    mr.onstop = () => {
      clearTimeout(timer);
      stream.getTracks().forEach((t) => t.stop());
      resolve({ blob: new Blob(chunks, { type: mr.mimeType || 'audio/webm' }), seconds: secondsOf(t0, Date.now()) });
    };
  });
  mr.start();
  timer = setTimeout(() => mr.state !== 'inactive' && mr.stop(), MAX_SECONDS * 1000);
  return { stop: () => mr.state !== 'inactive' && mr.stop(), done };
}
