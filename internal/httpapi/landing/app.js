// The page's own behaviour: theme, sprites, cards, terminal steps. The 3D world is
// loaded separately and is optional; everything here works without it.

import { LANG, setLanguage, tr, translatePage } from './i18n.js';

const root = document.documentElement;
const THEME_KEY = 'synagent.theme';
const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches;

/* ---------- Sprites: drawn as text, so they're crisp and follow the theme. ---------- */

const ICONS = {
  heart: ['.XX...XX.', 'XXXX.XXXX', 'XXXXXXXXX', 'XXXXXXXXX', '.XXXXXXX.', '..XXXXX..', '...XXX...', '....X....'],
  sun: ['....X....', '.X.....X.', '...XXX...', '..XXXXX..', 'X.XXXXX.X', '..XXXXX..', '...XXX...', '.X.....X.', '....X....'],
  moon: ['...XXXX..', '..XXX....', '.XXX.....', '.XXX.....', '.XXX.....', '.XXXX...X', '..XXXXXX.', '...XXXX..'],
  chat: ['XXXXXXXXX', 'X.......X', 'X.XX.XX.X', 'X.......X', 'X.XXXX..X', 'X.......X', 'XXXXXXXXX', '..XX.....', '..X......'],
  pin: ['..XXXXX..', '.XXXXXXX.', 'XXX...XXX', 'XX.....XX', 'XXX...XXX', '.XXXXXXX.', '..XXXXX..', '...XXX...', '....X....'],
  star: ['....X....', '....X....', '...XXX...', 'XXXXXXXXX', '.XXXXXXX.', '..XXXXX..', '.XXX.XXX.', '.XX...XX.', '.X.....X.'],
  shield: ['XXXXXXXXX', 'X.......X', 'X.....X.X', 'X....XX.X', 'XX..XX.XX', '.XXXX..X.', '..X...X..', '...X.X...', '....X....'],
  eye: ['.........', '..XXXXX..', '.XX...XX.', 'XX..X..XX', 'XX.XXX.XX', 'XX..X..XX', '.XX...XX.', '..XXXXX..'],
  trash: ['...XXX...', 'XXXXXXXXX', '.........', '.XXXXXXX.', '.X.X.X.X.', '.X.X.X.X.', '.X.X.X.X.', '.X.X.X.X.', '.XXXXXXX.'],
  box: ['XXXXXXXXX', 'X...X...X', 'XXXXXXXXX', 'X.......X', 'X..XXX..X', 'X.......X', 'XXXXXXXXX'],
  door: ['.XXXXXX..', '.X....X..', '.X....X.X', '.X....XXX', '.X..X.X.X', '.X....X..', '.X....X..', '.XXXXXX..'],
};

function svg(rows, colors, label) {
  const w = Math.max(...rows.map((r) => r.length));
  let rects = '';
  rows.forEach((row, y) => {
    for (let x = 0; x < row.length; x++) {
      const fill = colors[row[x]];
      if (fill) rects += `<rect x="${x}" y="${y}" width="1.02" height="1.02" fill="${fill}"/>`;
    }
  });
  const a11y = label ? `role="img" aria-label="${label}"` : 'aria-hidden="true"';
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${w} ${rows.length}" shape-rendering="crispEdges" ${a11y}>${rects}</svg>`;
}

for (const el of document.querySelectorAll('[data-sprite]')) {
  el.innerHTML = svg(ICONS[el.dataset.sprite], { X: 'currentColor' });
}

/* ---------- Theme ---------- */

let onTheme = () => {};

function saved() {
  try {
    return localStorage.getItem(THEME_KEY);
  } catch {
    return null;
  }
}

function applyTheme(theme) {
  root.dataset.theme = theme;
  document.querySelector('meta[name="theme-color"]').content = theme === 'light' ? '#f6f1ff' : '#05030c';
  document.getElementById('themeBtn').setAttribute('aria-label', tr(theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'));
  onTheme(theme);
}

applyTheme(root.dataset.theme);

document.getElementById('themeBtn').addEventListener('click', () => {
  const next = root.dataset.theme === 'light' ? 'dark' : 'light';
  try {
    localStorage.setItem(THEME_KEY, next);
  } catch {
    // Private windows: it still applies for this visit.
  }
  applyTheme(next);
});

// Until they've picked, follow the system, even if it changes at sunset.
matchMedia('(prefers-color-scheme: light)').addEventListener('change', (e) => {
  if (!saved()) applyTheme(e.matches ? 'light' : 'dark');
});

/* ---------- Dialogue box ---------- */

const LINES = [
  "* It's a good day to meet someone.",
  "* You'd be surprised what a real conversation can do.",
  "* I won't show you a deck. I'll just tell you about someone.",
  "* It's only a match when you both say yes.",
];
const typed = document.getElementById('typed');
const langSelect = document.getElementById('langSelect');
langSelect.value = LANG;
langSelect.addEventListener('change', () => setLanguage(langSelect.value));

function typeLine(el, line, then) {
  if (reduced) {
    el.textContent = line;
    if (then) then();
    return;
  }
  clearInterval(el._typing);
  let i = 0;
  el.textContent = '';
  el._typing = setInterval(() => {
    el.textContent = line.slice(0, ++i);
    if (i >= line.length) {
      clearInterval(el._typing);
      if (then) then();
    }
  }, 34);
}

if (!reduced) {
  let n = 0;
  const next = () => typeLine(typed, tr(LINES[n++ % LINES.length]), () => setTimeout(next, 3200));
  next();
}

/* ---------- Sample introductions. Words only, never a score. ---------- */

const PEOPLE = [
  {
    name: 'Mira', why: 'Climbs on Sundays, argues about ramen. You said you want someone curious.',
    colors: { H: '#b4532e', S: '#f2c6a4', E: '#27143f', M: '#b83a4b', B: '#e98f8f', T: '#3f9c8e' },
    art: ['....HHHHHHHH....', '...HHHHHHHHHH...', '..HHHHHHHHHHHH..', '..HHSSSSSSSSHH..', '..HSSSSSSSSSSH..', '..HSEESSSSEESH..', '..HSSSSSSSSSSH..', '..HSBSSMMSSBSH..', '..HHSSSSSSSSHH..', '..HHHSSSSSSHHH..', '..HH..SSSS..HH..', '..HHTTTTTTTTHH..', '.HHTTTTTTTTTTHH.', '.HTTTTTTTTTTTTH.', '.TTTTTTTTTTTTTT.', '.TTTTTTTTTTTTTT.'],
  },
  {
    name: 'Theo', why: 'Bakes badly, reads a lot. You both think a walk beats a bar.',
    colors: { H: '#2d2142', S: '#d9a27c', E: '#1a0f33', M: '#7a3238', G: '#f6f1ff', T: '#d98a2b', A: '#8c5716' },
    art: ['................', '....HHHHHHHH....', '...HHHHHHHHHH...', '...HHHHSHHHHH...', '...HSSSSSSSSH...', '..SSGGGSSGGGSS..', '...SGEGGGGEGS...', '...SSSSSSSSSS...', '...SSSMMMMSSS...', '....SSSSSSSS....', '......SSSS......', '....TTTTTTTT....', '..TTTTTTTTTTTT..', '.TTTTATTTTATTTT.', '.TTTTATTTTATTTT.', '.TTTTTTTTTTTTTT.'],
  },
  {
    name: 'Jun', why: 'Sings off-key on purpose. You told me you laugh at bad puns. Trust me on this.',
    colors: { H: '#17102b', S: '#f0cda6', E: '#17102b', M: '#c44361', B: '#ef9fa6', T: '#7a45f5', A: '#ff4163' },
    art: ['......HHHH......', '......HHHH......', '....AHHHHHHA....', '...HHHHHHHHHH...', '...HHSSSSSSHH...', '...HSSSSSSSSH...', '...HSEESSEESH...', '...SSSSSSSSSS...', '...SBSSMMSSBS...', '....SSSSSSSS....', '......SSSS......', '....TTTTTTTT....', '..TTTTTTTTTTTT..', '.TTTTTTTTTTTTTT.', '.TTTTTTTTTTTTTT.', '.TTTTTTTTTTTTTT.'],
  },
  {
    name: 'Noor', why: 'Draws maps of places that don\'t exist. You both want to travel slowly.',
    colors: { H: '#c94f7c', S: '#b97a56', E: '#1d0f22', M: '#6e2230', D: '#a73a64', T: '#f2e4c9', A: '#d8c49c' },
    art: ['.....HHHHHH.....', '...HHHHHHHHHH...', '..HHHHHHHHHHHH..', '..HHHSSSSSSHHH..', '..HHSSSSSSSSHH..', '..HHSEESSEESHH..', '..HHSSSSSSSSHH..', '..HHSSSMMSSSHH..', '..HHHSSSSSSHHH..', '..DHHHHHHHHHHD..', '..DDHHHHHHHHDD..', '...DDDDDDDDDD...', '..TTDDDDDDDDTT..', '.TTTTTTAATTTTTT.', '.TTTTTTAATTTTTT.', '.TTTTTTTTTTTTTT.'],
  },
  {
    name: 'Sam', why: 'Runs a tiny record shop. You both still buy things just for the sleeve art.',
    colors: { H: '#332014', S: '#7a4c30', E: '#120a06', M: '#3f1d1d', T: '#c63a3a', A: '#f6f1ff' },
    art: ['...HHHHHHHHHH...', '..HHHHHHHHHHHH..', '.HHHHHHHHHHHHHH.', '.HHHHHHHHHHHHHH.', '.HHHSSSSSSSSHHH.', '.HHSSSSSSSSSSHH.', '..HSEESSSSEESH..', '...SSSSSSSSSS...', '...SSSMMMMSSS...', '....SSSSSSSS....', '......SSSS......', '....TTAAAATT....', '..TTTTAAAATTTT..', '.TTTTTTTTTTTTTT.', '.TTTTTTTTTTTTTT.', '.TTTTTTTTTTTTTT.'],
  },
];
const FAN = [[-11, 44], [-5, 12], [0, 0], [5, 12], [11, 44]];
const fan = document.getElementById('fan');
const fanSay = document.getElementById('fanSay');

PEOPLE.forEach((p, i) => {
  const li = document.createElement('li');
  li.className = 'fan-card';
  li.style.setProperty('--r', `${FAN[i][0]}deg`);
  li.style.setProperty('--y', `${FAN[i][1]}px`);
  li.innerHTML = `<span class="fan-art">${svg(p.art, p.colors)}</span><h3>${p.name}</h3><p>${tr(p.why)}</p>
    <div class="fan-actions"><button type="button" class="no" aria-label="${tr('Pass on {name}', { name: p.name })}">X</button><button type="button" class="yes" aria-label="${tr('Say yes to {name}', { name: p.name })}"><i class="spr">${svg(ICONS.heart, { X: 'currentColor' })}</i></button></div>`;
  li.querySelector('.yes').addEventListener('click', () => {
    li.classList.remove('is-passed');
    li.classList.add('is-yes');
    typeLine(fanSay, tr("* Nice. If {name} says yes too, you'll start talking.", { name: p.name }));
  });
  li.querySelector('.no').addEventListener('click', () => {
    li.classList.remove('is-yes');
    li.classList.add('is-passed');
    typeLine(fanSay, tr("* Noted. Passing tells me something too. I'll learn from it."));
  });
  fan.append(li);
});

/* ---------- Terminal steps ---------- */

const STEPS = [
  { title: 'Get to know Suki', state: 'listening', body: 'Set up your profile by talking to her: who you are, your photos, your interests, and what you\'re looking for.' },
  { title: 'Get introduced', state: 'searching', body: 'She looks for people in your city who suit you, and explains why she thinks you\'d get along. It\'s only a match when you both say yes.' },
  { title: 'Start talking', state: 'coaching', body: 'Chat with your match directly. She can suggest an opener or a question when things go quiet, and you can turn that off.' },
  { title: 'Plan the date', state: 'scouting', body: 'She shortlists public places based on what you both enjoy. You each pick a favorite privately, and the date is set once you both confirm.' },
  { title: 'Tell her how it went', state: 'learning', body: 'What you share after a date helps her make better introductions next time.' },
];
const stepsEl = document.getElementById('steps');
const detail = document.getElementById('stepDetail');
const holoState = document.getElementById('holoState');
let current = -1;

function setStep(i) {
  if (i === current) return;
  current = i;
  stepsEl.querySelectorAll('button').forEach((b, n) => {
    if (n === i) b.setAttribute('aria-current', 'step');
    else b.removeAttribute('aria-current');
  });
  detail.innerHTML = `<h3>${tr('Step {n}', { n: i + 1 })} · ${tr(STEPS[i].title)}</h3><p>${tr(STEPS[i].body)}</p>`;
  holoState.textContent = tr(STEPS[i].state);
  document.getElementById('crtState').textContent = tr(STEPS[i].state);
}

let goToStep = setStep;
STEPS.forEach((s, i) => {
  const li = document.createElement('li');
  const b = document.createElement('button');
  b.type = 'button';
  b.innerHTML = `<b>${i + 1}</b><span>${tr(s.title)}</span>`;
  b.addEventListener('click', () => goToStep(i));
  li.append(b);
  stepsEl.append(li);
});
setStep(0);

translatePage();

/* ---------- The 3D world ---------- */

function flat() {
  root.classList.add('flat-crt');
}

const narrow = matchMedia('(max-width: 1080px)');
if (narrow.matches || reduced) flat();
narrow.addEventListener('change', (e) => {
  if (e.matches) flat();
  else if (!reduced && !root.classList.contains('no-webgl')) root.classList.remove('flat-crt');
});

import('./scene.js')
  .then(({ createWorld }) => {
    const world = createWorld({
      canvas: document.getElementById('world'),
      screen: document.getElementById('screen'),
      stage: document.getElementById('stage'),
      forge: document.querySelector('.forge'),
      forgeHead: document.getElementById('forgeHead'),
      finale: document.querySelector('.finale'),
      reduced,
      theme: root.dataset.theme,
      stepCount: STEPS.length,
      onStep: setStep,
    });
    onTheme = world.setTheme;
    goToStep = (i) => (root.classList.contains('flat-crt') ? setStep(i) : world.scrollToStep(i));
  })
  .catch((err) => {
    console.warn('3D world unavailable, using the flat page.', err);
    root.classList.add('no-webgl');
    flat();
  });
