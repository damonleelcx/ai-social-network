// Shapes from docs/03-api.md. Go encodes empty lists as null in places, so lists a
// screen depends on are normalized in endpoints.ts rather than trusted here.

export type User = {
  id: string;
  email: string;
  name: string;
  role: string;
  email_verified: boolean;
  has_password: boolean;
  created_at: string;
  admin: boolean;
};

export type Gender = 'woman' | 'man' | 'nonbinary' | 'other';
export type Prompt = { q: string; a: string };

export type Profile = {
  user_id: string;
  display_name: string;
  birthdate: string;
  age: number;
  gender: string;
  pronouns: string;
  seeking: string[] | null;
  city_key: string;
  city_name: string;
  age_min: number;
  age_max: number;
  goal: string;
  bio: string;
  bio_draft: string;
  occupation: string;
  education: string;
  height_cm: number | null;
  smoking: string;
  drinking: string;
  kids: string;
  wants_kids: string;
  religion: string;
  politics: string;
  languages: string[] | null;
  interests: string[] | null;
  prompts: Prompt[] | null;
  dealbreakers: Record<string, string[]> | null;
  status: 'onboarding' | 'active' | 'paused' | 'held' | string;
};

export type Me = { user: User; profile?: Profile; missing?: string[] };

export type PhotoStatus = 'pending' | 'approved' | 'rejected' | 'review';
export type Photo = {
  id: string;
  width: number;
  height: number;
  status: PhotoStatus;
  reason: string;
  tips: string;
  is_primary: boolean;
};

export type Mood = 'happy' | 'surprised' | 'thinking' | 'playful' | 'pouty';

export type Card =
  | { kind: 'photo_upload' }
  | { kind: 'bio_draft'; text: string }
  | { kind: 'proposal'; proposal_id: string }
  | { kind: 'match'; match_id: string }
  | { kind: 'date'; date_id: string; focus?: 'time' | 'change' }
  | { kind: 'share_plan'; date_id: string }
  | { kind: 'open_screen'; screen: string; label: string }
  | { kind: 'buddy_post'; activity: string; when: string; note?: string; expires_in_hours: number; same_gender?: boolean }
  | { kind: 'buddy_request'; request_id: string }
  | { kind: 'buy_credits' }
  | { kind: 'buddy_offer'; request_id: string; user_id: string }
  | {
      kind: 'confirm';
      action: 'remove_photo' | 'unmatch' | 'block' | 'reset_taste' | 'buddy_close';
      label: string;
      photo_id?: string;
      match_id?: string;
      user_id?: string;
      request_id?: string;
    };

export type MessageMeta = {
  mood?: Mood;
  replies?: string[];
  cards?: Card[];
  date_id?: string;
  card?: 'vote' | 'time' | 'confirm' | 'confirmed';
  at?: string;
};

export type Message = {
  thread_id: string;
  seq: number;
  sender_id?: string;
  role: 'user' | 'assistant' | 'event' | 'text' | 'prompt' | 'date' | 'system' | string;
  body: string;
  meta?: MessageMeta;
  read_at?: string;
  created_at: string;
};

export type Person = {
  user_id: string;
  display_name: string;
  age: number;
  gender: string;
  pronouns: string;
  city_name: string;
  goal: string;
  bio: string;
  occupation: string;
  education: string;
  height_cm: number | null;
  smoking: string;
  drinking: string;
  kids: string;
  wants_kids: string;
  languages: string[] | null;
  interests: string[] | null;
  prompts: Prompt[] | null;
  photo_ids: string[] | null;
};

export type Proposal = {
  id: string;
  status: string;
  pitch: string;
  shared: string[] | null;
  my_decision: string;
  person: Person;
  created_at: string;
  expires_at: string;
  match_id?: string;
};

export type Match = {
  id: string;
  stage: string;
  status: string;
  person: Person;
  created_at: string;
  last_message_at: string | null;
  last_preview: string;
  unread: number;
  open_date_id?: string;
  /** The kind of plan this match came from. Absent for an introduction. */
  activity?: string;
};

export type CreditPackage = { id: string; credits: number; amount_cents: number; currency: string };
export type CreditStatus = {
  balance: number;
  matches_used: number;
  free_matches: number;
  free: boolean;
  cost_per_message: number;
  payments_enabled: boolean;
  packages: CreditPackage[];
  google_enabled: boolean;
  google_products: { id: string; credits: number }[];
  google_account: string;
};

export interface ReferralStatus {
  code: string;
  link: string;
  credits_per_friend: number;
  joined: number;
  rewarded: number;
  credits_earned: number;
}

export type CreditUsage = { delta: number; reason: string; created_at: string };

export type BuddyRequest = {
  id: string;
  kind: string;
  note: string;
  when: string;
  city_name: string;
  same_gender: boolean;
  status: 'open' | 'filled' | 'closed';
  match_id?: string;
  created_at: string;
  expires_at: string;
  offered: boolean;
  person?: Person;
  offers?: { person: Person; created_at: string }[];
};

export type Hint = { id: string; body: string; suggestions: string[] | null; created_at: string };

export type Venue = {
  id: string;
  name: string;
  kind: string;
  address: string;
  lat: number;
  lon: number;
  website?: string;
  hours?: string;
  reason?: string;
};

export type DateStatus = 'searching' | 'voting' | 'scheduling' | 'confirmed' | 'completed' | 'cancelled' | 'failed';

export type DatePlan = {
  id: string;
  match_id: string;
  status: DateStatus;
  venues: Venue[] | null;
  my_vote: string;
  other_voted: boolean;
  venue: Venue | null;
  constraints: string[] | null;
  proposed_at: string | null;
  proposed_by_me: boolean;
  scheduled_at: string | null;
  person: Person;
  created_at: string;
};

export type Settings = {
  language: 'en' | 'zh' | 'ko' | 'ja' | 'es';
  theme: 'system' | 'light' | 'dark';
  notify_matches: boolean;
  notify_messages: boolean;
  notify_dates: boolean;
  notify_suki: boolean;
  notify_email: boolean;
  read_receipts: boolean;
  daily_matches: number;
  suki_tone: 'playful' | 'warm' | 'direct' | 'calm';
  suki_emoji: 'none' | 'some' | 'lots';
  suki_proactive: 'low' | 'normal' | 'high';
  suki_nickname: string;
  mood_avatar: boolean;
  chat_coach: boolean;
  chat_coach_read: boolean;
  chat_prompts: boolean;
  memory_enabled: boolean;
  debrief_enabled: boolean;
  suki_voice: boolean;
  suki_voice_auto: boolean;
};

export type Session = {
  id: string;
  user_agent: string;
  ip: string;
  created_at: string;
  last_seen_at: string;
  current: boolean;
};

export type Memory = { id: string; note: string; created_at: string };
export type Blocked = { user_id: string; display_name: string };

export type NotificationData = { screen?: 'agent' | 'proposal' | 'chat' | 'date'; id?: string };
export type AppNotification = {
  id: number;
  kind: string;
  title: string;
  body: string;
  data: NotificationData | null;
  read: boolean;
  created_at: string;
};

export type ReportReason = 'harassment' | 'fake_profile' | 'inappropriate_content' | 'scam' | 'underage' | 'other';

/** Crowds only: areas about 500 km across, with rounded counts. */
export type UserMap = { total: number; cells: { lat: number; lon: number; n: number }[] };
