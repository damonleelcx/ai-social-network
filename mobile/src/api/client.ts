import Constants from 'expo-constants';

import { tr } from '@/i18n';

const DEFAULT_API = 'https://match.heros-agent.space';

export const API_URL: string = (
  process.env.EXPO_PUBLIC_API_URL ||
  (Constants.expoConfig?.extra as { apiUrl?: string } | undefined)?.apiUrl ||
  DEFAULT_API
).replace(/\/+$/, '');

export const OFFLINE_MESSAGE = "I can't reach the server right now. Check your connection and try again.";
const GENERIC_MESSAGE = 'Something went wrong on our side. Please try again.';
const NEEDS_VERIFY = 'please verify your email first';
const SIGNED_OUT = 'please sign in';

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    // Every failure ends up on screen through here, the server's words included.
    super(tr(message));
  }
}

type Handlers = { onSignedOut: () => void; onNeedsVerify: () => void };

let token: string | null = null;
let handlers: Handlers = { onSignedOut: () => {}, onNeedsVerify: () => {} };

export function setToken(t: string | null) {
  token = t;
}
export function getToken() {
  return token;
}
export function setApiHandlers(h: Handlers) {
  handlers = h;
}

export function authHeaders(): Record<string, string> {
  return token ? { Authorization: `Bearer ${token}` } : {};
}

export type RequestOptions = {
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE';
  body?: unknown;
  timeoutMs?: number;
  /**
   * This call checks a password, so a 401 can mean "wrong password" and not
   * "signed out". The session middleware's own 401 says "please sign in", and
   * only that one ends the session here.
   */
  checksPassword?: boolean;
};

export async function api<T = unknown>(path: string, opts: RequestOptions = {}): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), opts.timeoutMs ?? 20000);
  const sentWith = token;
  let res: Response;
  try {
    res = await fetch(API_URL + path, {
      method: opts.method ?? 'GET',
      signal: controller.signal,
      headers: {
        Accept: 'application/json',
        ...(opts.body !== undefined ? { 'Content-Type': 'application/json' } : {}),
        ...authHeaders(),
      },
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    });
  } catch {
    throw new ApiError(0, OFFLINE_MESSAGE);
  } finally {
    clearTimeout(timer);
  }

  const text = await res.text().catch(() => '');
  let data: any = undefined;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = undefined;
    }
  }
  if (res.ok) return data as T;
  throw handleFailure(res.status, typeof data?.error === 'string' ? data.error : '', sentWith, opts.checksPassword);
}

/** Maps a failed response to an ApiError, and ends the session or asks for verification if that's what it means. */
export function handleFailure(status: number, message: string, sentWith: string | null, checksPassword?: boolean) {
  if (status === 401 && sentWith && sentWith === token && (!checksPassword || message === SIGNED_OUT)) {
    handlers.onSignedOut();
  }
  // 403 is also what an underage birthdate gets, so this matches the exact message.
  if (status === 403 && message === NEEDS_VERIFY) handlers.onNeedsVerify();
  return new ApiError(status, message || GENERIC_MESSAGE);
}

export function mediaSource(id: string) {
  return { uri: `${API_URL}/v1/media/${id}`, headers: authHeaders(), cacheKey: `media-${id}` };
}

export function socketUrl() {
  return API_URL.replace(/^http/, 'ws') + '/v1/ws';
}

export function legalUrl(page: 'terms' | 'privacy' | 'delete-account') {
  return `${API_URL}/legal/${page}`;
}
