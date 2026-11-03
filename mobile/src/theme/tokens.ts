// The whole visual language, from Suki's character sheet: lilac and violet,
// iridescent holographic accents, frosted panels, soft cyan and pink highlights,
// rounded pills and a dark violet ink.

export type Tokens = {
  scheme: 'light' | 'dark';
  /** Page background: a slow diagonal drift through lilac with a hint of pink and cyan. */
  bg: [string, string, string, string];
  ink: string;
  inkSoft: string;
  inkFaint: string;
  glass: string;
  glassStrong: string;
  glassBorder: string;
  /** Sheets sit over busy screens, so they're opaque. */
  sheet: string;
  hairline: string;
  accent: string;
  accentSoft: string;
  /** Filled controls and my own bubbles. White text sits on this. */
  brand: [string, string, string];
  /** The holographic shimmer: rings, rules, focus. */
  iris: [string, string, string];
  onBrand: string;
  bubbleOther: string;
  cardTint: string;
  scrim: string;
  good: string;
  goodSoft: string;
  warn: string;
  warnSoft: string;
  bad: string;
  badSoft: string;
  statusBar: 'light' | 'dark';
};

export const light: Tokens = {
  scheme: 'light',
  bg: ['#F6F1FF', '#EDE5FF', '#F7E9F8', '#E8F3FC'],
  ink: '#2A1A4F',
  inkSoft: '#5B4A82',
  inkFaint: '#85759F',
  glass: 'rgba(255,255,255,0.62)',
  glassStrong: 'rgba(255,255,255,0.92)',
  glassBorder: 'rgba(255,255,255,0.95)',
  sheet: '#FBF8FF',
  hairline: 'rgba(92,60,170,0.14)',
  accent: '#6D3BEF',
  accentSoft: 'rgba(122,69,245,0.12)',
  brand: ['#7A45F5', '#A04CE6', '#D45CBE'],
  iris: ['#8BE8F5', '#B59CFF', '#F6A6DE'],
  onBrand: '#FFFFFF',
  bubbleOther: 'rgba(255,255,255,0.86)',
  cardTint: 'rgba(181,156,255,0.16)',
  scrim: 'rgba(32,18,70,0.42)',
  good: '#177F68',
  goodSoft: 'rgba(43,169,138,0.14)',
  warn: '#96660B',
  warnSoft: 'rgba(214,160,40,0.16)',
  bad: '#C22F5B',
  badSoft: 'rgba(214,69,107,0.12)',
  statusBar: 'dark',
};

export const dark: Tokens = {
  scheme: 'dark',
  bg: ['#150B30', '#1E1144', '#2A124F', '#14183F'],
  ink: '#F4EEFF',
  inkSoft: '#C8BAEA',
  inkFaint: '#9A8BC2',
  glass: 'rgba(255,255,255,0.07)',
  glassStrong: 'rgba(44,28,92,0.97)',
  glassBorder: 'rgba(205,180,255,0.20)',
  sheet: '#2A1B59',
  hairline: 'rgba(205,180,255,0.16)',
  accent: '#BFA2FF',
  accentSoft: 'rgba(181,156,255,0.16)',
  brand: ['#7A45F5', '#A04CE6', '#D45CBE'],
  iris: ['#7FDDEB', '#B59CFF', '#F19AD6'],
  onBrand: '#FFFFFF',
  bubbleOther: 'rgba(255,255,255,0.10)',
  cardTint: 'rgba(181,156,255,0.10)',
  scrim: 'rgba(5,2,20,0.62)',
  good: '#6FE0C3',
  goodSoft: 'rgba(111,224,195,0.14)',
  warn: '#F0C568',
  warnSoft: 'rgba(240,197,104,0.14)',
  bad: '#FF8FAE',
  badSoft: 'rgba(255,143,174,0.14)',
  statusBar: 'light',
};

export const space = { xs: 4, sm: 8, md: 12, lg: 16, xl: 20, xxl: 28, huge: 40 };
export const radius = { sm: 12, md: 18, lg: 24, xl: 32, pill: 999 };

export const type = {
  display: { fontSize: 36, lineHeight: 42, fontWeight: '800', letterSpacing: -0.6 },
  title: { fontSize: 26, lineHeight: 32, fontWeight: '800', letterSpacing: -0.3 },
  heading: { fontSize: 19, lineHeight: 25, fontWeight: '700' },
  subheading: { fontSize: 16, lineHeight: 22, fontWeight: '700' },
  body: { fontSize: 16, lineHeight: 23, fontWeight: '400' },
  small: { fontSize: 14, lineHeight: 20, fontWeight: '400' },
  caption: { fontSize: 12, lineHeight: 16, fontWeight: '500' },
  label: { fontSize: 12, lineHeight: 16, fontWeight: '700', letterSpacing: 0.6, textTransform: 'uppercase' },
} as const;

export type TypeName = keyof typeof type;
