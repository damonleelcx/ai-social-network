import type {
  AppNotification,
  Blocked,
  BuddyRequest,
  Person,
  GroupChat,
  GroupMessage,
  Trip,
  CreditStatus,
  CreditUsage,
  ReferralStatus,
  DatePlan,
  Hint,
  Match,
  Me,
  Memory,
  Message,
  Photo,
  Profile,
  Proposal,
  ReportReason,
  Session,
  Settings,
  User,
  UserMap,
} from './types';
import { currentLanguage } from '@/i18n';
import { api } from './client';

type AuthResult = { token: string; expires_at: string; user: User };
type Ok = { ok: boolean };

/** Open to everyone, signed in or not. */
export const publicInfo = {
  userMap: () => api<UserMap>('/v1/public/user-map', { timeoutMs: 8000 }),
};

export const auth = {
  signUp: (b: { email: string; password: string; name: string; accept_terms: boolean; language?: string; ref?: string }) =>
    api<AuthResult>('/v1/auth/signup', { method: 'POST', body: b }),
  signIn: (email: string, password: string) =>
    api<AuthResult>('/v1/auth/signin', { method: 'POST', body: { email, password }, checksPassword: true }),
  google: (id_token: string, accept_terms: boolean, ref?: string) =>
    api<AuthResult>('/v1/auth/google', { method: 'POST', body: { id_token, accept_terms, ref, language: currentLanguage() }, checksPassword: true }),
  deleteWithGoogle: (id_token: string) => api<Ok>('/v1/auth/google/delete', { method: 'POST', body: { id_token }, timeoutMs: 60000 }),
  signOut: () => api<Ok>('/v1/auth/signout', { method: 'POST', body: {} }),
  verify: (code: string) => api<{ user: User }>('/v1/auth/verify', { method: 'POST', body: { code } }),
  resend: () => api<Ok>('/v1/auth/resend', { method: 'POST', body: {} }),
  forgot: (email: string) => api<Ok>('/v1/auth/forgot', { method: 'POST', body: { email } }),
  reset: (email: string, code: string, password: string) =>
    api<Ok>('/v1/auth/reset', { method: 'POST', body: { email, code, password } }),
  changePassword: (current: string, next: string) =>
    api<Ok>('/v1/auth/change-password', { method: 'POST', body: { current, next }, checksPassword: true }),
  sessions: async () => (await api<{ sessions: Session[] | null }>('/v1/auth/sessions')).sessions ?? [],
  revokeSession: (id: string) => api<Ok>(`/v1/auth/sessions/${id}`, { method: 'DELETE' }),
  deleteAccount: (password: string) =>
    api<Ok>('/v1/account/delete', { method: 'POST', body: { password }, checksPassword: true, timeoutMs: 60000 }),
  exportData: () => api<unknown>('/v1/account/export', { timeoutMs: 60000 }),
  me: () => api<Me>('/v1/me'),
};

export const profile = {
  get: () => api<{ profile: Profile; missing: string[] | null }>('/v1/profile'),
  patch: async (patch: Record<string, unknown>) =>
    (await api<{ profile: Profile }>('/v1/profile', { method: 'PATCH', body: patch })).profile,
  setCity: async (query: string) =>
    (await api<{ city_name: string }>('/v1/profile/city', { method: 'POST', body: { query } })).city_name,
  pause: (paused: boolean) => api<{ paused: boolean }>('/v1/profile/pause', { method: 'POST', body: { paused } }),
  acceptBio: () => api<Ok>('/v1/profile/bio-draft/accept', { method: 'POST', body: {} }),
};

export const photos = {
  list: async () => (await api<{ photos: Photo[] | null }>('/v1/photos')).photos ?? [],
  setPrimary: (id: string) => api<Ok>(`/v1/photos/${id}/primary`, { method: 'POST', body: {} }),
  remove: (id: string) => api<Ok>(`/v1/photos/${id}`, { method: 'DELETE' }),
};

export const suki = {
  speechStatus: () => api<{ enabled: boolean }>('/v1/agent/speech'),
  messages: (after = 0) =>
    api<{ messages: Message[] | null; thinking: boolean }>(`/v1/agent/messages${after > 0 ? `?after=${after}` : ''}`),
  send: (text: string, client_id: string) =>
    api<{ message: Message }>('/v1/agent/messages', { method: 'POST', body: { text, client_id } }),
  opened: () => api<Ok>('/v1/agent/opened', { method: 'POST', body: {} }),
};

export const proposals = {
  list: async () => (await api<{ proposals: Proposal[] | null }>('/v1/proposals')).proposals ?? [],
  get: async (id: string) => (await api<{ proposal: Proposal }>(`/v1/proposals/${id}`)).proposal,
  decide: async (id: string, decision: 'accept' | 'pass') =>
    (await api<{ match_id: string }>(`/v1/proposals/${id}/decide`, { method: 'POST', body: { decision } })).match_id,
};

export const matches = {
  list: async () => (await api<{ matches: Match[] | null }>('/v1/matches')).matches ?? [],
  get: async (id: string) => (await api<{ match: Match }>(`/v1/matches/${id}`)).match,
  unmatch: (id: string) => api<Ok>(`/v1/matches/${id}/unmatch`, { method: 'POST', body: {} }),
  messages: async (id: string, after = 0) =>
    (await api<{ messages: Message[] | null }>(`/v1/matches/${id}/messages${after > 0 ? `?after=${after}` : ''}`)).messages ?? [],
  send: async (id: string, text: string, client_id: string) =>
    (await api<{ message: Message }>(`/v1/matches/${id}/messages`, { method: 'POST', body: { text, client_id } })).message,
  read: (id: string, seq: number) => api<Ok>(`/v1/matches/${id}/read`, { method: 'POST', body: { seq } }),
  typing: (id: string) => api<Ok>(`/v1/matches/${id}/typing`, { method: 'POST', body: {} }),
  hints: async (id: string) => (await api<{ hints: Hint[] | null }>(`/v1/matches/${id}/hints`)).hints ?? [],
  coach: (id: string) => api<Ok>(`/v1/matches/${id}/coach`, { method: 'POST', body: {} }),
  dismissHint: (hintId: string) => api<Ok>(`/v1/hints/${hintId}/dismiss`, { method: 'POST', body: {} }),
  startDate: async (id: string, constraints: string[]) =>
    (await api<{ date_id: string }>(`/v1/matches/${id}/date`, { method: 'POST', body: { constraints } })).date_id,
};

type DateResult = { date: DatePlan };
export const dates = {
  list: async () => (await api<{ dates: DatePlan[] | null }>('/v1/dates')).dates ?? [],
  get: async (id: string) => (await api<DateResult>(`/v1/dates/${id}`)).date,
  vote: async (id: string, venue_id: string) =>
    (await api<DateResult>(`/v1/dates/${id}/vote`, { method: 'POST', body: { venue_id } })).date,
  time: async (id: string, at: string) => (await api<DateResult>(`/v1/dates/${id}/time`, { method: 'POST', body: { at } })).date,
  confirm: async (id: string) => (await api<DateResult>(`/v1/dates/${id}/confirm`, { method: 'POST', body: {} })).date,
  cancel: async (id: string, reason?: string) =>
    (await api<DateResult>(`/v1/dates/${id}/cancel`, { method: 'POST', body: { reason: reason ?? '' } })).date,
  share: async (id: string) => (await api<{ text: string }>(`/v1/dates/${id}/share`)).text,
};

export const referral = {
  status: () => api<ReferralStatus>('/v1/referral'),
};

export const credits = {
  status: () => api<CreditStatus>('/v1/credits'),
  usage: async () => (await api<{ usage: CreditUsage[] }>('/v1/credits/usage')).usage,
  verifyGoogle: (product_id: string, purchase_token: string) =>
    api<{ pending: boolean }>('/v1/credits/google/verify', { method: 'POST', body: { product_id, purchase_token } }),
  checkout: async (pack: string) => (await api<{ url: string }>('/v1/credits/checkout', { method: 'POST', body: { package: pack } })).url,
};

export const buddies = {
  board: async () => {
    const r = await api<{ requests: BuddyRequest[] | null; missing: string[] | null }>('/v1/buddies');
    return { requests: r.requests ?? [], missing: r.missing ?? [] };
  },
  mine: async () => (await api<{ requests: BuddyRequest[] | null }>('/v1/buddies/mine')).requests ?? [],
  get: async (id: string) => (await api<{ request: BuddyRequest }>(`/v1/buddies/${id}`)).request,
  post: async (b: {
    kind: string;
    when: string;
    note: string;
    expires_in_hours: number;
    same_gender: boolean;
    spots?: number;
    place?: string;
    trip_id?: string;
  }) =>
    (await api<{ request: BuddyRequest }>('/v1/buddies', { method: 'POST', body: b })).request,
  offer: (id: string) => api<Ok>(`/v1/buddies/${id}/offer`, { method: 'POST', body: {} }),
  withdraw: (id: string) => api<Ok>(`/v1/buddies/${id}/offer`, { method: 'DELETE' }),
  pick: async (id: string, user_id: string) =>
    (await api<{ match_id: string }>(`/v1/buddies/${id}/pick`, { method: 'POST', body: { user_id } })).match_id,
  close: (id: string) => api<Ok>(`/v1/buddies/${id}/close`, { method: 'POST', body: {} }),
};

/** What someone wrote about themselves, translated. Only the fields they filled in. */
export type PersonWords = Partial<Pick<Person, 'bio' | 'city_name' | 'occupation' | 'education' | 'interests' | 'prompts'>>;

export const people = {
  translation: (id: string, lang: string) =>
    api<{ lang: string; translated: boolean; words: PersonWords }>(`/v1/people/${id}/translation?lang=${encodeURIComponent(lang)}`),
};

export type LookingFor = {
  must: { seeking: string[]; age_min?: number; age_max?: number; goal?: string; city?: string };
  red_flags: Record<string, string[]>;
  nice: { id: string; note: string }[];
  read?: LookingLine[];
};

export type LookingLine = { id: string; kind: 'toward' | 'away'; text: string; said: string };

export const lookingFor = {
  get: () => api<LookingFor>('/v1/looking-for'),
  dropRedFlag: (field: string) => api<Ok>(`/v1/looking-for/red-flags/${field}`, { method: 'DELETE' }),
  dropRead: (id: string) => api<Ok>(`/v1/looking-for/read/${id}`, { method: 'DELETE' }),
};

export const groups = {
  mine: async () => (await api<{ requests: BuddyRequest[] | null }>('/v1/groups')).requests ?? [],
  chat: async (id: string, after = 0) => {
    const r = await api<GroupChat>(`/v1/groups/${id}/messages${after ? `?after=${after}` : ''}`);
    return { ...r, messages: r.messages ?? [] };
  },
  send: async (id: string, text: string, client_id: string) =>
    (await api<{ message: GroupMessage }>(`/v1/groups/${id}/messages`, { method: 'POST', body: { text, client_id } })).message,
};

export const trips = {
  mine: async () => (await api<{ trips: Trip[] | null }>('/v1/trips')).trips ?? [],
  get: async (id: string) => (await api<{ trip: Trip }>(`/v1/trips/${id}`)).trip,
  add: async (b: { city: string; starts_on: string; ends_on: string; note?: string }) =>
    (await api<{ trip: Trip }>('/v1/trips', { method: 'POST', body: b })).trip,
  remove: (id: string) => api<Ok>(`/v1/trips/${id}`, { method: 'DELETE' }),
};

export const safety = {
  blocked: async () => (await api<{ blocked: Blocked[] | null }>('/v1/blocks')).blocked ?? [],
  block: (user_id: string) => api<Ok>('/v1/blocks', { method: 'POST', body: { user_id } }),
  unblock: (id: string) => api<Ok>(`/v1/blocks/${id}`, { method: 'DELETE' }),
  report: (b: { user_id: string; match_id?: string; message_seq?: number; reason: ReportReason; details?: string }) =>
    api<Ok>('/v1/reports', { method: 'POST', body: b }),
};

export const settings = {
  get: async () => (await api<{ settings: Settings }>('/v1/settings')).settings,
  patch: async (patch: Partial<Settings>) =>
    (await api<{ settings: Settings }>('/v1/settings', { method: 'PATCH', body: patch })).settings,
};

export const memories = {
  list: async () => (await api<{ memories: Memory[] | null }>('/v1/memories')).memories ?? [],
  remove: (id: string) => api<Ok>(`/v1/memories/${id}`, { method: 'DELETE' }),
  reset: () => api<Ok>('/v1/memories/reset', { method: 'POST', body: {} }),
};

export const notifications = {
  list: async () => (await api<{ notifications: AppNotification[] | null }>('/v1/notifications')).notifications ?? [],
  markRead: () => api<Ok>('/v1/notifications/read', { method: 'POST', body: {} }),
  registerPush: (token: string, platform: string) =>
    api<Ok>('/v1/push-tokens', { method: 'POST', body: { token, platform } }),
};
