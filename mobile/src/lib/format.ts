
import { dateLocale, tr } from '@/i18n';

export function uuid() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
  });
}

const sameDay = (a: Date, b: Date) =>
  a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();

// Dates are spelled out here rather than by toLocaleDateString: Android's JavaScript engine often
// has no locale data, and then writes "Saturday, November 14" in a Chinese app.
type Names = { days: string[]; short: string[]; months: string[]; monthsShort: string[] };
const NAMES: Record<string, Names> = {
  zh: {
    days: ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六'],
    short: ['周日', '周一', '周二', '周三', '周四', '周五', '周六'],
    months: [],
    monthsShort: [],
  },
  ja: {
    days: ['日曜日', '月曜日', '火曜日', '水曜日', '木曜日', '金曜日', '土曜日'],
    short: ['日', '月', '火', '水', '木', '金', '土'],
    months: [],
    monthsShort: [],
  },
  ko: {
    days: ['일요일', '월요일', '화요일', '수요일', '목요일', '금요일', '토요일'],
    short: ['일', '월', '화', '수', '목', '금', '토'],
    months: [],
    monthsShort: [],
  },
  es: {
    days: ['domingo', 'lunes', 'martes', 'miércoles', 'jueves', 'viernes', 'sábado'],
    short: ['dom', 'lun', 'mar', 'mié', 'jue', 'vie', 'sáb'],
    months: ['enero', 'febrero', 'marzo', 'abril', 'mayo', 'junio', 'julio', 'agosto', 'septiembre', 'octubre', 'noviembre', 'diciembre'],
    monthsShort: ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sept', 'oct', 'nov', 'dic'],
  },
};
const lang = () => dateLocale().slice(0, 2);
const two = (n: number) => String(n).padStart(2, '0');

export function clock(d: Date) {
  const h = d.getHours();
  const m = two(d.getMinutes());
  switch (lang()) {
    case 'zh':
      return `${h < 12 ? '上午' : '下午'}${h % 12 || 12}:${m}`;
    case 'ko':
      return `${h < 12 ? '오전' : '오후'} ${h % 12 || 12}:${m}`;
    case 'ja':
    case 'es':
      return `${h}:${m}`;
    default:
      return d.toLocaleTimeString(dateLocale(), { hour: 'numeric', minute: '2-digit' });
  }
}

/** "Mon", in the app's language. */
function weekdayShort(d: Date) {
  const n = NAMES[lang()];
  if (!n) return d.toLocaleDateString(dateLocale(), { weekday: 'short' });
  return lang() === 'ja' ? `(${n.short[d.getDay()]})` : n.short[d.getDay()];
}

/** "Mar 3", in the app's language. */
export function monthDay(d: Date) {
  const m = d.getMonth() + 1;
  const day = d.getDate();
  switch (lang()) {
    case 'zh':
    case 'ja':
      return `${m}月${day}日`;
    case 'ko':
      return `${m}월 ${day}일`;
    case 'es':
      return `${day} ${NAMES.es.monthsShort[m - 1]}`;
    default:
      return d.toLocaleDateString(dateLocale(), { month: 'short', day: 'numeric' });
  }
}

/** "3:04 PM" today, "Mon 3:04 PM" this week, otherwise "Mar 3". For message lists. */
export function shortTime(iso?: string | null) {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const now = new Date();
  if (sameDay(d, now)) return clock(d);
  if (now.getTime() - d.getTime() < 6 * 86400000) return `${weekdayShort(d)} ${clock(d)}`;
  return monthDay(d);
}

/** "Saturday, March 3", in the app's language. */
function longDay(d: Date) {
  const m = d.getMonth() + 1;
  const day = d.getDate();
  const n = NAMES[lang()];
  switch (lang()) {
    case 'zh':
      return `${m}月${day}日 ${n.days[d.getDay()]}`;
    case 'ja':
      return `${m}月${day}日 ${n.days[d.getDay()]}`;
    case 'ko':
      return `${m}월 ${day}일 ${n.days[d.getDay()]}`;
    case 'es':
      return `${n.days[d.getDay()]}, ${day} de ${n.months[m - 1]}`;
    default:
      return d.toLocaleDateString(dateLocale(), { weekday: 'long', month: 'long', day: 'numeric' });
  }
}

/** "Saturday, March 3 at 7:30 PM", for dates. */
export function longDateTime(iso?: string | null) {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return tr('{day} at {value}', { day: longDay(d), value: clock(d) });
}

export function fullDate(ymd: string) {
  const d = new Date(`${ymd}T12:00:00`);
  if (Number.isNaN(d.getTime())) return ymd;
  const y = d.getFullYear();
  const m = d.getMonth() + 1;
  const day = d.getDate();
  switch (lang()) {
    case 'zh':
    case 'ja':
      return `${y}年${m}月${day}日`;
    case 'ko':
      return `${y}년 ${m}월 ${day}일`;
    case 'es':
      return `${day} de ${NAMES.es.months[m - 1]} de ${y}`;
    default:
      return d.toLocaleDateString(dateLocale(), { year: 'numeric', month: 'long', day: 'numeric' });
  }
}

export function ymd(d: Date) {
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
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
  tr(list.find((x) => x.value === v)?.label ?? '');

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
    display_name: tr('Your name'),
    birthdate: tr('Your birthday'),
    gender: tr('Your gender'),
    seeking: tr('Who you want to meet'),
    city: tr('Your city'),
    goal: tr("What you're looking for"),
    'interests (at least 3)': tr('A few interests'),
    'bio or a prompt answer': tr('A bio or a prompt answer'),
    photo: tr('A photo'),
    'photo (waiting for review)': tr('Your photo is being checked'),
  };
  return known[m] ?? m.replace(/_/g, ' ').replace(/^./, (c) => c.toUpperCase());
}
