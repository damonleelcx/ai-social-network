import { catalog } from './catalog';

// Copy is written in English and looked up by its English text, so a string with no translation
// just shows in English. Nothing reads the phone's language: English until someone chooses.
const COLUMN = { zh: 0, ko: 1, ja: 2 } as const;
const LOCALE = { en: 'en', zh: 'zh-CN', ko: 'ko-KR', ja: 'ja-JP', es: 'es' } as const;

export type Language = keyof typeof LOCALE;

export const LANGUAGE_NAMES: readonly { value: Language; label: string }[] = [
  { value: 'en', label: 'English' },
  { value: 'zh', label: '中文' },
  { value: 'ko', label: '한국어' },
  { value: 'ja', label: '日本語' },
];

let current: Language = 'en';

export const isLanguage = (v: unknown): v is Language => typeof v === 'string' && v in LOCALE;

/** Callers remount the screens afterwards: copy is read as they render, and tables at import time stay in English. */
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
