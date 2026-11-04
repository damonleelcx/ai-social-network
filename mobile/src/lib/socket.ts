import { Platform } from 'react-native';
import { getToken, socketUrl } from '@/api/client';

/** "reconnect" is ours: the socket came back, so everything may be stale. */
export type Topic = string;
type Listener = (topic: Topic, data?: unknown) => void;

const listeners = new Set<Listener>();

/** The small event bus. Screens subscribe here (or with useTopic), never to the socket. */
export const bus = {
  on(listener: Listener) {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  },
  emit(topic: Topic, data?: unknown) {
    listeners.forEach((l) => l(topic, data));
  },
};

const MIN_BACKOFF = 1000;
const MAX_BACKOFF = 30000;

let ws: WebSocket | null = null;
let wanted = false;
let attempt = 0;
let everOpened = false;
let timer: ReturnType<typeof setTimeout> | null = null;

function clearTimer() {
  if (timer) clearTimeout(timer);
  timer = null;
}

function scheduleReconnect() {
  clearTimer();
  if (!wanted) return;
  const base = Math.min(MAX_BACKOFF, MIN_BACKOFF * 2 ** attempt);
  attempt += 1;
  timer = setTimeout(open, base * (0.7 + Math.random() * 0.6));
}

function open() {
  clearTimer();
  const token = getToken();
  if (!wanted || !token || ws) return;
  // React Native's third argument; the DOM lib's type doesn't know about it.
  const Native = WebSocket as unknown as new (url: string, protocols: null, options: { headers: Record<string, string> }) => WebSocket;
  const socket =
    Platform.OS === 'web'
      ? new WebSocket(socketUrl(), ['synagent', `bearer.${token}`])
      : new Native(socketUrl(), null, { headers: { Authorization: `Bearer ${token}` } });
  ws = socket;
  socket.onopen = () => {
    attempt = 0;
    if (everOpened) bus.emit('reconnect');
    everOpened = true;
  };
  socket.onmessage = (e) => {
    try {
      const { topic, data } = JSON.parse(String(e.data));
      if (typeof topic === 'string') bus.emit(topic, data);
    } catch {
      // Not a frame we understand; the next refetch covers it.
    }
  };
  socket.onerror = () => {
    // onclose always follows.
  };
  socket.onclose = () => {
    if (ws === socket) ws = null;
    scheduleReconnect();
  };
}

export const socket = {
  /** Safe to call repeatedly: a live socket is left alone. */
  connect() {
    wanted = true;
    if (!ws) {
      attempt = 0;
      open();
    }
  },
  disconnect() {
    wanted = false;
    clearTimer();
    const current = ws;
    ws = null;
    current?.close();
  },
  /** Back in the foreground the caller refetches, so the next open needn't announce itself. */
  reset() {
    everOpened = false;
  },
};
