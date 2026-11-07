import { catalog } from './catalog';

// Copy is written in English and looked up by its English text, so a string with no translation
// just shows in English. Until someone chooses, the phone's language decides (deviceLanguage).
const COLUMN = { zh: 0, ko: 1, ja: 2, es: 3 } as const;
const LOCALE = { en: 'en', zh: 'zh-CN', ko: 'ko-KR', ja: 'ja-JP', es: 'es' } as const;

export type Language = keyof typeof LOCALE;

export const LANGUAGE_NAMES: readonly { value: Language; label: string }[] = [
  { value: 'en', label: 'English' },
  { value: 'zh', label: '中文' },
  { value: 'ko', label: '한국어' },
  { value: 'ja', label: '日本語' },
  { value: 'es', label: 'Español' },
];

let current: Language = 'en';

export const isLanguage = (v: unknown): v is Language => typeof v === 'string' && v in LOCALE;

/** The first of the device's preferred languages that we have, else English. */
export function deviceLanguage(): Language {
  const tags: string[] = [];
  const nav = (globalThis as { navigator?: { languages?: readonly string[]; language?: string } }).navigator;
  if (nav?.languages?.length) tags.push(...nav.languages);
  else if (nav?.language) tags.push(nav.language);
  try {
    tags.push(Intl.DateTimeFormat().resolvedOptions().locale);
  } catch {
    // No Intl: English.
  }
  for (const tag of tags) {
    const base = tag.toLowerCase().split(/[-_]/)[0];
    if (base !== 'en' && isLanguage(base)) return base;
    if (base === 'en') return 'en';
  }
  return 'en';
}

/** Copy is read as screens render, and the session re-renders them. Tables built at import time would stay in English. */
export function setLanguage(language: Language) {
  current = language;
}

export const currentLanguage = () => current;

/** For dates and times, so they match the words around them. */
export const dateLocale = () => LOCALE[current];

export function tr(english: string, vars?: Record<string, string | number>) {
  const column = COLUMN[current as keyof typeof COLUMN];
  let text = column === undefined ? english : (catalog[english]?.[column] ?? english);
  for (const [k, v] of Object.entries(vars ?? {})) text = text.replace(`{${k}}`, String(v));
  return text;
}
