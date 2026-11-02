import { Image } from 'expo-image';
import * as SecureStore from 'expo-secure-store';
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { ApiError, setApiHandlers, setToken } from '@/api/client';
import { auth, settings as settingsApi } from '@/api/endpoints';
import type { Me, Profile, Settings, User } from '@/api/types';
import { socket } from '@/lib/socket';
import { voice } from '@/lib/voice';

const TOKEN_KEY = 'synagent.token';
const THEME_KEY = 'synagent.theme';
export const SEEN_KEY = 'synagent.suki.seen';

export type SessionStatus = 'loading' | 'unreachable' | 'signedOut' | 'unverified' | 'ready';

type SessionValue = {
  status: SessionStatus;
  user: User | null;
  profile: Profile | null;
  missing: string[];
  settings: Settings | null;
  cachedTheme: Settings['theme'] | null;
  /** Stores a freshly issued token and routes by /v1/me. */
  adoptToken: (token: string) => Promise<void>;
  refreshMe: () => Promise<void>;
  /** Retries the first load after a network failure. */
  retry: () => void;
  signOut: () => Promise<void>;
  /** Clears everything local. For after the account or this session is gone. */
  forget: () => Promise<void>;
  updateSettings: (patch: Partial<Settings>) => Promise<void>;
};

const Ctx = createContext<SessionValue | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<SessionStatus>('loading');
  const [me, setMe] = useState<Me | null>(null);
  const [settings, setSettings] = useState<Settings | null>(null);
  const [cachedTheme, setCachedTheme] = useState<Settings['theme'] | null>(null);
  const alive = useRef(true);

  const forget = useCallback(async () => {
    setToken(null);
    socket.disconnect();
    socket.reset();
    voice.clearCache();
    setMe(null);
    setSettings(null);
    setStatus('signedOut');
    await Promise.allSettled([
      SecureStore.deleteItemAsync(TOKEN_KEY),
      SecureStore.deleteItemAsync(SEEN_KEY),
      // Other people's photos shouldn't outlive the session on this device.
      Image.clearDiskCache(),
      Image.clearMemoryCache(),
    ]);
  }, []);

  const applyMe = useCallback((next: Me) => {
    setMe(next);
    setStatus(next.user.email_verified ? 'ready' : 'unverified');
  }, []);

  const refreshMe = useCallback(async () => {
    const next = await auth.me();
    if (alive.current) applyMe(next);
  }, [applyMe]);

  const loadSettings = useCallback(async () => {
    try {
      const s = await settingsApi.get();
      setSettings(s);
      SecureStore.setItemAsync(THEME_KEY, s.theme).catch(() => {});
    } catch {
      // Screens that need settings show their own state; the theme stays as it was.
    }
  }, []);

  const boot = useCallback(async () => {
    setStatus('loading');
    let stored: string | null = null;
    try {
      stored = await SecureStore.getItemAsync(TOKEN_KEY);
    } catch {
      stored = null;
    }
    if (!stored) {
      setStatus('signedOut');
      return;
    }
    setToken(stored);
    try {
      const next = await auth.me();
      applyMe(next);
      loadSettings();
    } catch (e) {
      // A 401 already went through forget(). Anything else, keep the token and offer a retry.
      if (!(e instanceof ApiError && e.status === 401)) setStatus('unreachable');
    }
  }, [applyMe, loadSettings]);

  useEffect(() => {
    alive.current = true;
    setApiHandlers({
      onSignedOut: () => {
        forget();
      },
      onNeedsVerify: () => setStatus('unverified'),
    });
    SecureStore.getItemAsync(THEME_KEY)
      .then((t) => {
        if (t === 'light' || t === 'dark' || t === 'system') setCachedTheme(t);
      })
      .catch(() => {});
    boot();
    return () => {
      alive.current = false;
    };
  }, [boot, forget]);

  const adoptToken = useCallback(
    async (token: string) => {
      setToken(token);
      // If the keychain refuses, they stay signed in until the app closes. Better than
      // turning a correct password into a technical error.
      await SecureStore.setItemAsync(TOKEN_KEY, token).catch(() => {});
      const next = await auth.me();
      applyMe(next);
      loadSettings();
    },
    [applyMe, loadSettings],
  );

  const signOut = useCallback(async () => {
    // Best effort: if the server can't hear it, the session still ends here.
    await auth.signOut().catch(() => {});
    await forget();
  }, [forget]);

  const updateSettings = useCallback(
    async (patch: Partial<Settings>) => {
      const before = settings;
      if (before) setSettings({ ...before, ...patch });
      try {
        const s = await settingsApi.patch(patch);
        setSettings(s);
        SecureStore.setItemAsync(THEME_KEY, s.theme).catch(() => {});
      } catch (e) {
        if (before) setSettings(before);
        throw e;
      }
    },
    [settings],
  );

  const value = useMemo<SessionValue>(
    () => ({
      status,
      user: me?.user ?? null,
      profile: me?.profile ?? null,
      missing: me?.missing ?? [],
      settings,
      cachedTheme,
      adoptToken,
      refreshMe,
      retry: boot,
      signOut,
      forget,
      updateSettings,
    }),
    [status, me, settings, cachedTheme, adoptToken, refreshMe, boot, signOut, forget, updateSettings],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useSession() {
  const v = useContext(Ctx);
  if (!v) throw new Error('useSession outside SessionProvider');
  return v;
}
