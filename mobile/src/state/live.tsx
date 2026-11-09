import * as SecureStore from '@/lib/store';
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { AppState } from 'react-native';
import * as endpoints from '@/api/endpoints';
import type { AppNotification, DatePlan, Match, Message, Photo, Proposal } from '@/api/types';
import { uuid } from '@/lib/format';
import { bus, socket } from '@/lib/socket';
import { voice } from '@/lib/voice';
import { SEEN_KEY, useSession } from './session';

const POLL_MS = 4000;
const SUKI_TYPING_MS = 30000;
/** A stream that goes quiet this long lost its final message somewhere; stop showing it. */
const STREAM_STALE_MS = 45000;
const STREAM_CHECK_MS = 5000;

/** One reply as it's being written. Best-effort: the stored message is always the truth. */
export type SukiStream = {
  turn: string;
  text: string;
  /** The newest assistant message when this began. A newer one means it's finished. */
  since: number;
};

/** A chunk arriving: restart on reset, otherwise take only the piece that continues the text. */
export function applyStreamChunk(cur: SukiStream[], chunk: { turn: string; offset: number; text: string; reset: boolean }, since: number) {
  const i = cur.findIndex((s) => s.turn === chunk.turn);
  const existing = i >= 0 ? cur[i] : { turn: chunk.turn, text: '', since };
  let text: string;
  if (chunk.reset) text = '';
  else if (chunk.offset === existing.text.length) text = existing.text + chunk.text;
  else return cur;
  const next = { ...existing, text };
  return i >= 0 ? cur.map((s, j) => (j === i ? next : s)) : [...cur, next];
}

export type PendingMessage = {
  client_id: string;
  text: string;
  state: 'sending' | 'failed';
  error?: string;
  created_at: string;
};

export function mergeBySeq(a: Message[], b: Message[]) {
  const map = new Map<number, Message>();
  for (const m of a) map.set(m.seq, m);
  for (const m of b) map.set(m.seq, m);
  return [...map.values()].sort((x, y) => x.seq - y.seq);
}

type LiveValue = {
  proposals: Proposal[];
  matches: Match[];
  dates: DatePlan[];
  notifications: AppNotification[];
  photos: Photo[];
  loaded: { proposals: boolean; matches: boolean; dates: boolean; notifications: boolean; photos: boolean; suki: boolean };
  refreshProposals: () => Promise<void>;
  refreshMatches: () => Promise<void>;
  refreshDates: () => Promise<void>;
  refreshNotifications: () => Promise<void>;
  refreshPhotos: () => Promise<void>;

  sukiMessages: Message[];
  sukiPending: PendingMessage[];
  sukiThinking: boolean;
  sukiTyping: boolean;
  /** A reply is on its way over the stream. Its text is in useSukiStreams. */
  sukiStreaming: boolean;
  sukiError: string | null;
  refreshSuki: (full?: boolean) => Promise<void>;
  sendToSuki: (text: string) => void;
  retryToSuki: (clientId: string) => void;
  discardPending: (clientId: string) => void;
  /** The Suki tab is on screen, so what she's said counts as seen. */
  setSukiFocused: (focused: boolean) => void;

  badges: { suki: number; matches: number; dates: number; me: number };
};

const Ctx = createContext<LiveValue | null>(null);
// Separate, because it changes with every word and only the bubble needs to know.
const StreamCtx = createContext<SukiStream[]>([]);

/** A date is waiting on this person if there's something only they can do. */
export function dateNeedsMe(d: DatePlan) {
  if (d.status === 'voting') return !d.my_vote;
  if (d.status === 'scheduling') return !d.proposed_at || !d.proposed_by_me;
  return false;
}

/** Mounted only while signed in and verified, so everything resets on sign-out. */
export function LiveProvider({ children }: { children: ReactNode }) {
  const { refreshMe } = useSession();

  const [proposals, setProposals] = useState<Proposal[]>([]);
  const [matches, setMatches] = useState<Match[]>([]);
  const [dates, setDates] = useState<DatePlan[]>([]);
  const [notifications, setNotifications] = useState<AppNotification[]>([]);
  const [photos, setPhotos] = useState<Photo[]>([]);
  const [loaded, setLoaded] = useState<LiveValue['loaded']>({
    proposals: false,
    matches: false,
    dates: false,
    notifications: false,
    photos: false,
    suki: false,
  });
  const markLoaded = useCallback((k: keyof LiveValue['loaded']) => {
    setLoaded((l) => (l[k] ? l : { ...l, [k]: true }));
  }, []);

  const [sukiMessages, setSukiMessages] = useState<Message[]>([]);
  const [sukiPending, setSukiPending] = useState<PendingMessage[]>([]);
  const [sukiThinking, setSukiThinking] = useState(false);
  const [sukiTyping, setSukiTyping] = useState(false);
  const [sukiError, setSukiError] = useState<string | null>(null);
  const [seenSeq, setSeenSeq] = useState<number | null>(null);
  const [streams, setStreams] = useState<SukiStream[]>([]);
  const streamsRef = useRef(streams);
  streamsRef.current = streams;
  const lastAssistantSeq = useRef(0);
  const lastChunkAt = useRef(0);
  // Turns already finished, so a late chunk can't bring back a bubble.
  const finishedTurns = useRef(new Set<string>());
  const sukiFocused = useRef(false);
  const lastSeq = useRef(0);
  const typingTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const refreshProposals = useCallback(async () => {
    try {
      setProposals(await endpoints.proposals.list());
    } catch {
      // Keep what's on screen; the next event or foreground tries again.
    } finally {
      markLoaded('proposals');
    }
  }, [markLoaded]);
  const refreshMatches = useCallback(async () => {
    try {
      setMatches(await endpoints.matches.list());
    } catch {
    } finally {
      markLoaded('matches');
    }
  }, [markLoaded]);
  const refreshDates = useCallback(async () => {
    try {
      setDates(await endpoints.dates.list());
    } catch {
    } finally {
      markLoaded('dates');
    }
  }, [markLoaded]);
  const refreshNotifications = useCallback(async () => {
    try {
      setNotifications(await endpoints.notifications.list());
    } catch {
    } finally {
      markLoaded('notifications');
    }
  }, [markLoaded]);
  const refreshPhotos = useCallback(async () => {
    try {
      setPhotos(await endpoints.photos.list());
    } catch {
    } finally {
      markLoaded('photos');
    }
  }, [markLoaded]);

  const markSeen = useCallback((seq: number) => {
    setSeenSeq((cur) => {
      if (cur !== null && cur >= seq) return cur;
      SecureStore.setItemAsync(SEEN_KEY, String(seq)).catch(() => {});
      return seq;
    });
  }, []);

  const clearTyping = useCallback(() => {
    if (typingTimer.current) clearTimeout(typingTimer.current);
    typingTimer.current = null;
    setSukiTyping(false);
  }, []);

  const refreshSuki = useCallback(
    async (full = false) => {
      try {
        const r = await endpoints.suki.messages(full ? 0 : lastSeq.current);
        const incoming = r.messages ?? [];
        if (incoming.length) {
          lastSeq.current = Math.max(lastSeq.current, ...incoming.map((m) => m.seq));
          setSukiMessages((cur) => mergeBySeq(cur, incoming));
          const newestAssistant = incoming.reduce((n, m) => (m.role === 'assistant' ? Math.max(n, m.seq) : n), 0);
          if (newestAssistant > lastAssistantSeq.current) lastAssistantSeq.current = newestAssistant;
          // In the same update as the message above, so the bubble hands over in one frame.
          const finished = streamsRef.current.filter((s) => newestAssistant > s.since);
          if (finished.length) {
            finished.forEach((s) => finishedTurns.current.add(s.turn));
            setStreams((cur) => cur.filter((s) => !finishedTurns.current.has(s.turn)));
          }
        }
        setSukiThinking(r.thinking);
        setSukiError(null);
        if (sukiFocused.current && lastSeq.current > 0) markSeen(lastSeq.current);
      } catch (e: any) {
        setSukiError(e?.message ?? null);
      } finally {
        markLoaded('suki');
      }
    },
    [markLoaded, markSeen],
  );

  const deliver = useCallback(
    async (client_id: string, text: string) => {
      setSukiPending((p) => p.map((x) => (x.client_id === client_id ? { ...x, state: 'sending', error: undefined } : x)));
      try {
        const { message } = await endpoints.suki.send(text, client_id);
        lastSeq.current = Math.max(lastSeq.current, message.seq);
        setSukiMessages((cur) => mergeBySeq(cur, [message]));
        setSukiPending((p) => p.filter((x) => x.client_id !== client_id));
        setSukiThinking(true);
      } catch (e: any) {
        setSukiPending((p) => p.map((x) => (x.client_id === client_id ? { ...x, state: 'failed', error: e?.message } : x)));
      }
    },
    [],
  );

  const sendToSuki = useCallback(
    (raw: string) => {
      const text = raw.trim();
      if (!text) return;
      const client_id = uuid();
      setSukiPending((p) => [...p, { client_id, text, state: 'sending', created_at: new Date().toISOString() }]);
      deliver(client_id, text);
    },
    [deliver],
  );

  const pendingRef = useRef(sukiPending);
  pendingRef.current = sukiPending;
  const retryToSuki = useCallback(
    (clientId: string) => {
      const m = pendingRef.current.find((x) => x.client_id === clientId);
      if (m) deliver(m.client_id, m.text);
    },
    [deliver],
  );
  const discardPending = useCallback((clientId: string) => {
    setSukiPending((p) => p.filter((x) => x.client_id !== clientId));
  }, []);

  const setSukiFocused = useCallback(
    (focused: boolean) => {
      sukiFocused.current = focused;
      if (focused && lastSeq.current > 0) markSeen(lastSeq.current);
    },
    [markSeen],
  );

  const onStreamChunk = useCallback((data: unknown) => {
    const d = data as { turn?: unknown; offset?: unknown; text?: unknown; reset?: unknown } | null | undefined;
    if (typeof d?.turn !== 'string' || finishedTurns.current.has(d.turn)) return;
    const chunk = {
      turn: d.turn,
      offset: typeof d.offset === 'number' ? d.offset : -1,
      text: typeof d.text === 'string' ? d.text : '',
      reset: d.reset === true,
    };
    lastChunkAt.current = Date.now();
    setStreams((cur) => applyStreamChunk(cur, chunk, lastAssistantSeq.current));
  }, []);

  const refreshAll = useCallback(() => {
    refreshProposals();
    refreshMatches();
    refreshDates();
    refreshNotifications();
    refreshPhotos();
    refreshSuki(true);
    refreshMe().catch(() => {});
    voice.refreshEnabled();
  }, [refreshDates, refreshMatches, refreshMe, refreshNotifications, refreshPhotos, refreshProposals, refreshSuki]);

  // First load, the socket, and the app coming back to the foreground.
  useEffect(() => {
    SecureStore.getItemAsync(SEEN_KEY)
      .then((v) => setSeenSeq(v ? Number(v) || 0 : 0))
      .catch(() => setSeenSeq(0));
    refreshAll();
    socket.connect();
    endpoints.suki.opened().catch(() => {});
    const sub = AppState.addEventListener('change', (state) => {
      if (state === 'active') {
        socket.reset();
        socket.connect();
        refreshAll();
        endpoints.suki.opened().catch(() => {});
      } else if (state === 'background') {
        voice.stop();
        socket.disconnect();
      }
    });
    return () => {
      sub.remove();
      voice.stop();
      socket.disconnect();
      if (typingTimer.current) clearTimeout(typingTimer.current);
    };
  }, [refreshAll]);

  // What each topic means, per the contract.
  useEffect(
    () =>
      bus.on((topic, data) => {
        if (topic === 'reconnect') refreshAll();
        else if (topic === 'agent_stream') onStreamChunk(data);
        else if (topic === 'agent') {
          clearTyping();
          refreshSuki();
        } else if (topic === 'typing:agent') {
          setSukiTyping(true);
          if (typingTimer.current) clearTimeout(typingTimer.current);
          typingTimer.current = setTimeout(() => setSukiTyping(false), SUKI_TYPING_MS);
        } else if (topic === 'proposals') refreshProposals();
        else if (topic === 'matches' || topic.startsWith('chat:')) refreshMatches();
        else if (topic === 'dates') refreshDates();
        else if (topic === 'photos') {
          refreshPhotos();
          refreshMe().catch(() => {});
        } else if (topic === 'notification') refreshNotifications();
      }),
    [clearTyping, onStreamChunk, refreshAll, refreshDates, refreshMatches, refreshMe, refreshNotifications, refreshPhotos, refreshProposals, refreshSuki],
  );

  // The socket may be down, so don't rely on it to end "thinking".
  useEffect(() => {
    if (!sukiThinking) return;
    const id = setInterval(() => refreshSuki(), POLL_MS);
    return () => clearInterval(id);
  }, [refreshSuki, sukiThinking]);

  const sukiStreaming = streams.length > 0;

  useEffect(() => {
    if (!sukiStreaming) return;
    const id = setInterval(() => {
      if (Date.now() - lastChunkAt.current < STREAM_STALE_MS) return;
      streamsRef.current.forEach((s) => finishedTurns.current.add(s.turn));
      setStreams([]);
      refreshSuki();
    }, STREAM_CHECK_MS);
    return () => clearInterval(id);
  }, [refreshSuki, sukiStreaming]);

  const badges = useMemo(() => {
    const seen = seenSeq ?? Number.MAX_SAFE_INTEGER;
    return {
      suki: sukiMessages.filter((m) => m.role === 'assistant' && m.seq > seen).length,
      matches: proposals.length + matches.reduce((n, m) => n + (m.unread > 0 ? 1 : 0), 0),
      dates: dates.filter(dateNeedsMe).length,
      me: notifications.filter((n) => !n.read).length,
    };
  }, [dates, matches, notifications, proposals, seenSeq, sukiMessages]);

  const value = useMemo<LiveValue>(
    () => ({
      proposals,
      matches,
      dates,
      notifications,
      photos,
      loaded,
      refreshProposals,
      refreshMatches,
      refreshDates,
      refreshNotifications,
      refreshPhotos,
      sukiMessages,
      sukiPending,
      sukiThinking,
      sukiTyping,
      sukiStreaming,
      sukiError,
      refreshSuki,
      sendToSuki,
      retryToSuki,
      discardPending,
      setSukiFocused,
      badges,
    }),
    [
      proposals,
      matches,
      dates,
      notifications,
      photos,
      loaded,
      refreshProposals,
      refreshMatches,
      refreshDates,
      refreshNotifications,
      refreshPhotos,
      sukiMessages,
      sukiPending,
      sukiThinking,
      sukiTyping,
      sukiStreaming,
      sukiError,
      refreshSuki,
      sendToSuki,
      retryToSuki,
      discardPending,
      setSukiFocused,
      badges,
    ],
  );

  return (
    <Ctx.Provider value={value}>
      <StreamCtx.Provider value={streams}>{children}</StreamCtx.Provider>
    </Ctx.Provider>
  );
}

export function useSukiStreams() {
  return useContext(StreamCtx);
}

export function useLive() {
  const v = useContext(Ctx);
  if (!v) throw new Error('useLive outside LiveProvider');
  return v;
}

/** Calls back for socket topics. `match` picks them: an exact name, or a predicate. */
export function useTopic(match: string | ((topic: string) => boolean), cb: (topic: string) => void) {
  const latest = useRef(cb);
  latest.current = cb;
  const m = useRef(match);
  m.current = match;
  useEffect(
    () =>
      bus.on((t) => {
        const hit = typeof m.current === 'string' ? t === m.current || t === 'reconnect' : m.current(t) || t === 'reconnect';
        if (hit) latest.current(t);
      }),
    [],
  );
}

/** Refetch when the app returns to the foreground. */
export function useForeground(cb: () => void) {
  const latest = useRef(cb);
  latest.current = cb;
  useEffect(() => {
    const sub = AppState.addEventListener('change', (s) => {
      if (s === 'active') latest.current();
    });
    return () => sub.remove();
  }, []);
}
