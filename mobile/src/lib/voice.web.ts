import { useSyncExternalStore } from 'react';
import { API_URL, ApiError, authHeaders, getToken, handleFailure, OFFLINE_MESSAGE } from '@/api/client';
import { suki } from '@/api/endpoints';

/** voice.ts with the browser's Audio in place of the file cache and native player. */
export type ClipKey = number | 'sample';

type State = { enabled: boolean; clip: ClipKey | null; phase: 'loading' | 'playing' | null; tapped: boolean };

const REQUEST_MS = 60000;

let state: State = { enabled: false, clip: null, phase: null, tapped: false };
const listeners = new Set<() => void>();

function update(next: Partial<State>) {
  state = { ...state, ...next };
  listeners.forEach((l) => l());
}

let generation = 0;
let audio: HTMLAudioElement | null = null;
let abort: AbortController | null = null;

async function fetchClip(key: ClipKey, signal: AbortSignal) {
  const sentWith = getToken();
  let res: Response;
  try {
    res = await fetch(`${API_URL}/v1/agent/speech`, {
      method: 'POST',
      signal,
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify(key === 'sample' ? { sample: true } : { seq: key }),
    });
  } catch {
    throw new ApiError(0, OFFLINE_MESSAGE);
  }
  if (!res.ok) {
    const data = await res.json().catch(() => undefined);
    if (res.status === 503) update({ enabled: false });
    throw handleFailure(res.status, typeof data?.error === 'string' ? data.error : '', sentWith);
  }
  return URL.createObjectURL(await res.blob());
}

export const voice = {
  async refreshEnabled() {
    try {
      const { enabled } = await suki.speechStatus();
      update({ enabled: !!enabled });
    } catch {
      update({ enabled: false });
    }
  },

  stop() {
    generation += 1;
    abort?.abort();
    abort = null;
    if (audio) {
      audio.pause();
      URL.revokeObjectURL(audio.src);
      audio = null;
    }
    if (state.clip !== null) update({ clip: null, phase: null, tapped: false });
  },

  /** Browsers refuse sound nobody asked for, so an autoplay that's turned away just stops quietly. */
  async play(key: ClipKey, { tapped }: { tapped: boolean }) {
    voice.stop();
    const mine = ++generation;
    const controller = new AbortController();
    abort = controller;
    const timer = setTimeout(() => controller.abort(), REQUEST_MS);
    update({ clip: key, phase: 'loading', tapped });
    try {
      const url = await fetchClip(key, controller.signal);
      if (mine !== generation) {
        URL.revokeObjectURL(url);
        return;
      }
      const a = new Audio(url);
      audio = a;
      a.onended = () => {
        if (mine === generation) voice.stop();
      };
      update({ phase: 'playing' });
      await a.play().catch(() => {
        if (mine === generation) voice.stop();
      });
    } catch (e) {
      if (mine !== generation) return;
      voice.stop();
      throw e;
    } finally {
      clearTimeout(timer);
    }
  },

  clearCache() {
    voice.stop();
    update({ enabled: false });
  },

  current: () => state,
};

const subscribe = (l: () => void) => {
  listeners.add(l);
  return () => {
    listeners.delete(l);
  };
};
const snapshot = () => state;

export function useVoice() {
  return useSyncExternalStore(subscribe, snapshot);
}
