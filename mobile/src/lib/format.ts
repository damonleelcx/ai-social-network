export function uuid() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
  });
}

const sameDay = (a: Date, b: Date) =>
  a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();

export function clock(d: Date) {
  return d.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' });
}

/** "3:04 PM" today, "Mon 3:04 PM" this week, otherwise "Mar 3". For message lists. */
export function shortTime(iso?: string | null) {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const now = new Date();
  if (sameDay(d, now)) return clock(d);
  if (now.getTime() - d.getTime() < 6 * 86400000) {
    return `${d.toLocaleDateString(undefined, { weekday: 'short' })} ${clock(d)}`;
  }
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

/** "Saturday, March 3 at 7:30 PM", for dates. */
export function longDateTime(iso?: string | null) {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const day = d.toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' });
  return `${day} at ${clock(d)}`;
}

export function fullDate(ymd: string) {
  const d = new Date(`${ymd}T12:00:00`);
  if (Number.isNaN(d.getTime())) return ymd;
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' });
}

export function ymd(d: Date) {
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

export function deviceLanguage(): 'en' | 'zh' | 'ko' | 'ja' | 'es' {
  try {
    const l = Intl.DateTimeFormat().resolvedOptions().locale.slice(0, 2).toLowerCase();
    if (l === 'zh' || l === 'ko' || l === 'ja' || l === 'es') return l;
  } catch {
    // Fall through to English.
  }
  return 'en';
}

// ── Labels for the contract's enums. Words only, never a number. ───────────────

export const GENDERS = [
  { value: 'woman', label: 'Woman' },
  { value: 'man', label: 'Man' },
  { value: 'nonbinary', label: 'Nonbinary' },
  { value: 'other', label: 'Other' },
];
export const GOALS = [
  { value: 'long_term', label: 'Long-term' },
  { value: 'marriage', label: 'Marriage' },
  { value: 'short_term', label: 'Short-term' },
  { value: 'casual', label: 'Casual' },
  { value: 'friends_first', label: 'Friends first' },
  { value: 'not_sure', label: 'Not sure yet' },
];
export const HABITS = [
  { value: 'no', label: 'No' },
  { value: 'sometimes', label: 'Sometimes' },
  { value: 'yes', label: 'Yes' },
];
export const KIDS = [
  { value: 'none', label: "Don't have kids" },
  { value: 'have', label: 'Have kids' },
];
export const WANTS_KIDS = [
  { value: 'yes', label: 'Want kids' },
  { value: 'no', label: "Don't want kids" },
  { value: 'open', label: 'Open to it' },
];
export const REPORT_REASONS = [
  { value: 'harassment', label: 'Harassment' },
  { value: 'fake_profile', label: 'Fake profile' },
  { value: 'inappropriate_content', label: 'Inappropriate content' },
  { value: 'scam', label: 'Scam' },
  { value: 'underage', label: 'Under 18' },
  { value: 'other', label: 'Something else' },
] as const;

export const labelOf = (list: readonly { value: string; label: string }[], v?: string | null) =>
  list.find((x) => x.value === v)?.label ?? '';

export const STAGE_LABEL: Record<string, string> = {
  icebreaker: 'Just matched',
  getting_to_know: 'Getting to know each other',
  date_ready: 'Ready for a date',
  planning: 'Planning a date',
  planned: 'Date planned',
  post_date: 'After your date',
};

/** Turns the server's `missing` entries into words for a person. */
export function missingLabel(m: string) {
  const known: Record<string, string> = {
    display_name: 'Your name',
    birthdate: 'Your birthday',
    gender: 'Your gender',
    seeking: 'Who you want to meet',
    city: 'Your city',
    goal: "What you're looking for",
    'interests (at least 3)': 'A few interests',
    'bio or a prompt answer': 'A bio or a prompt answer',
    photo: 'A photo',
    'photo (waiting for review)': 'Your photo is being checked',
  };
  return known[m] ?? m.replace(/_/g, ' ').replace(/^./, (c) => c.toUpperCase());
}
