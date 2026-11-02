import { createAudioPlayer, setAudioModeAsync, type AudioPlayer } from 'expo-audio';
import { fetch as binaryFetch } from 'expo/fetch';
import { Directory, File, Paths } from 'expo-file-system';
import { useSyncExternalStore } from 'react';
import { API_URL, ApiError, authHeaders, getToken, handleFailure, OFFLINE_MESSAGE } from '@/api/client';
import { suki } from '@/api/endpoints';

/** A message's seq, or the fixed greeting from Settings. */
export type ClipKey = number | 'sample';

type State = {
  /** Does the server have a voice at all? Everything voice-related hides when it doesn't. */
  enabled: boolean;
  clip: ClipKey | null;
  phase: 'loading' | 'playing' | null;
  /** The person asked for this one, so autoplay must not talk over it. */
  tapped: boolean;
};

const MAX_CACHED = 60;
const REQUEST_MS = 60000;

let state: State = { enabled: false, clip: null, phase: null, tapped: false };
const listeners = new Set<() => void>();

function update(next: Partial<State>) {
  state = { ...state, ...next };
  listeners.forEach((l) => l());
}

let generation = 0;
let player: AudioPlayer | null = null;
let abort: AbortController | null = null;

function cacheDir() {
  return new Directory(Paths.cache, 'suki-voice');
}

function prune(dir: Directory) {
  const entries = dir.list();
  if (entries.length < MAX_CACHED) return;
  entries.forEach((e) => {
    try {
      e.delete();
    } catch {
      // A leftover clip is harmless.
    }
  });
}

async function clipFile(key: ClipKey, signal: AbortSignal) {
  const dir = cacheDir();
  if (!dir.exists) dir.create({ intermediates: true, idempotent: true });
  const file = new File(dir, key === 'sample' ? 'sample.mp3' : `${key}.mp3`);
  if (file.exists && file.size > 0) return file;

  const sentWith = getToken();
  let res: Awaited<ReturnType<typeof binaryFetch>>;
  try {
    res = await binaryFetch(`${API_URL}/v1/agent/speech`, {
      method: 'POST',
      signal,
      headers: { 'Content-Type': 'application/json', ...authHeaders() },
      body: JSON.stringify(key === 'sample' ? { sample: true } : { seq: key }),
    });
  } catch {
    throw new ApiError(0, OFFLINE_MESSAGE);
  }
  if (!res.ok) {
    let message = '';
    try {
      const data = JSON.parse(await res.text());
      if (typeof data?.error === 'string') message = data.error;
    } catch {
      // No JSON body; handleFailure supplies the generic line.
    }
    // The server says voice is off, so stop offering it.
    if (res.status === 503) update({ enabled: false });
    throw handleFailure(res.status, message, sentWith);
  }
  const bytes = new Uint8Array(await res.arrayBuffer());
  prune(dir);
  file.create({ overwrite: true });
  file.write(bytes);
  return file;
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
    const p = player;
    player = null;
    try {
      p?.remove();
    } catch {
      // Already released.
    }
    if (state.clip !== null) update({ clip: null, phase: null, tapped: false });
  },

  /**
   * Plays one clip, replacing whatever was playing. `tapped` means the person asked,
   * so it sounds even with the iOS silent switch on. Autoplay respects the switch.
   * Rejects with the API's error, unless a newer call or stop() took over.
   */
  async play(key: ClipKey, { tapped }: { tapped: boolean }) {
    voice.stop();
    const mine = ++generation;
    const controller = new AbortController();
    abort = controller;
    const timer = setTimeout(() => controller.abort(), REQUEST_MS);
    update({ clip: key, phase: 'loading', tapped });
    try {
      const file = await clipFile(key, controller.signal);
      if (mine !== generation) return;
      await setAudioModeAsync({ playsInSilentMode: tapped, shouldPlayInBackground: false, allowsRecording: false });
      if (mine !== generation) return;
      const p = createAudioPlayer({ uri: file.uri });
      player = p;
      p.addListener('playbackStatusUpdate', (s) => {
        if (s.didJustFinish && mine === generation) voice.stop();
      });
      p.play();
      update({ phase: 'playing' });
    } catch (e) {
      if (mine !== generation) return;
      voice.stop();
      throw e;
    } finally {
      clearTimeout(timer);
    }
  },

  /** Clips are the person's own conversation, so they don't outlive the session. */
  clearCache() {
    voice.stop();
    try {
      const dir = cacheDir();
      if (dir.exists) dir.delete();
    } catch {
      // The OS clears its cache eventually.
    }
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
