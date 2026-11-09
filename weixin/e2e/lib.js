// Helpers for driving the mini program in WeChat DevTools. Local only: it refuses any server
// that isn't on this machine, and takes no secrets. See README.md.
const automator = require('miniprogram-automator');
const crypto = require('crypto');
const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');

const WS = process.env.WX_WS || 'ws://127.0.0.1:9420';
const API = process.env.WX_API || 'http://127.0.0.1:8080';
const PG_CONTAINER = process.env.WX_PG_CONTAINER || 'syn-pg';
const PG_DB = process.env.WX_PG_DB || 'syn_live';
const OUT = process.env.WX_OUT || path.join(__dirname, 'out');

if (!/^(http|ws):\/\/(127\.0\.0\.1|localhost)(:\d+)?/.test(API)) throw new Error(`Refusing a server that isn't local: ${API}`);
if (!/^ws:\/\/(127\.0\.0\.1|localhost)(:\d+)?/.test(WS)) throw new Error(`Refusing a DevTools that isn't local: ${WS}`);
fs.mkdirSync(OUT, { recursive: true });

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const results = [];
const problems = [];
let m = null;
let shots = 0;

function check(ok, what, detail) {
  const line = `${ok ? 'PASS' : 'FAIL'}  ${what}${!ok && detail !== undefined ? ` -> ${String(detail).slice(0, 300)}` : ''}`;
  console.log(line);
  results.push({ ok: !!ok, what });
  return !!ok;
}

function note(text) {
  console.log(`NOTE  ${text}`);
}

async function connect() {
  m = await automator.connect({ wsEndpoint: WS });
  // A run that stopped early can leave a mock behind. Recompile in DevTools to clear it:
  // restoreWxMethod on something that isn't mocked wipes the real wx method.
  m.on('console', (c) => {
    const text = (c.args || []).map(String).join(' ') || c.text || '';
    if (c.type === 'error' || c.level === 'error') {
      problems.push(`console.error: ${text.slice(0, 300)}`);
      console.log(`CONSOLE-ERROR ${text.slice(0, 300)}`);
    }
  });
  m.on('exception', (e) => {
    problems.push(`exception: ${e.message}`);
    console.log(`EXCEPTION ${e.message}\n${(e.stack || '').slice(0, 500)}`);
  });
  return m;
}

async function page() {
  return m.currentPage();
}

async function waitUntil(fn, what, ms = 15000, every = 300) {
  const until = Date.now() + ms;
  let last;
  while (Date.now() < until) {
    try {
      last = await fn();
      if (last) return last;
    } catch (e) {
      last = e;
    }
    await sleep(every);
  }
  throw new Error(`timed out waiting for ${what} (last: ${last && last.message ? last.message : JSON.stringify(last)})`);
}

const waitPage = (p, ms) => waitUntil(async () => {
  const c = await page();
  return c && c.path === p ? c : null;
}, `page ${p}`, ms);

const waitData = (p, fn, what, ms) => waitUntil(async () => {
  const d = await p.data();
  return fn(d) ? d : null;
}, what, ms);

async function shot(name) {
  shots += 1;
  const file = path.join(OUT, `${Date.now().toString(36)}-${String(shots).padStart(3, '0')}-${name}.png`);
  await m.screenshot({ path: file });
  return file;
}

// The first element matching the selector whose text contains these words.
async function byText(p, selector, text, ms = 8000) {
  return waitUntil(async () => {
    for (const el of await p.$$(selector)) {
      if ((await el.text()).includes(text)) return el;
    }
    return null;
  }, `${selector} saying "${text}"`, ms);
}

async function tapText(p, selector, text, ms) {
  const el = await byText(p, selector, text, ms);
  await el.tap();
  await sleep(250);
  return el;
}

async function fill(p, selector, value, index = 0) {
  const list = await waitUntil(async () => {
    const l = await p.$$(selector);
    return l.length > index ? l : null;
  }, `input ${selector}`);
  await list[index].input(value);
  await sleep(80);
}

// DevTools sometimes drops a reLaunch while it's busy, so this tries again.
async function goto(url) {
  for (let i = 0; ; i++) {
    try {
      await m.reLaunch(url);
      break;
    } catch (e) {
      if (i >= 2) throw e;
      note(`reLaunch retry: ${e.message}`);
      await sleep(3000);
    }
  }
  await sleep(700);
  return page();
}

// Through the real button, because the token also lives in the app's memory.
async function signOut() {
  if (!(await m.callWxMethod('getStorageSync', 'token'))) return;
  await goto('/pages/settings/account/account');
  // A token the server has dropped sends the app to login by itself.
  const state = await waitUntil(async () => {
    const c = await page();
    if (c.path === 'pages/login/login') return { gone: true };
    const d = await c.data();
    return d.user ? { page: c, d } : null;
  }, 'account page');
  if (state.gone) {
    note('The stored token was no longer valid; the app went to login by itself.');
    return;
  }
  await tapText(state.page, 'button', state.d.t.signOut);
  await waitPage('pages/login/login');
}

async function signIn(account) {
  await signOut();
  const p = await goto('/pages/login/login');
  const d = await waitData(p, (x) => x.t && x.t.useEmail, 'login page');
  await tapText(p, '.link', d.t.useEmail);
  await fill(p, 'input[data-f=email]', account.email);
  await fill(p, 'input[data-f=password]', account.password);
  await tapText(p, 'button', d.t.signIn);
  return waitPage('pages/suki/suki', 20000);
}

// Says something to Suki through the composer, and waits for her to finish answering.
async function sukiSay(p, text, ms = 150000) {
  const before = (await p.data()).messages.length;
  await fill(p, '.composer input', text);
  await (await p.$('.composer .send')).tap();
  return waitData(p, (d) => {
    const last = d.messages[d.messages.length - 1];
    return d.messages.length > before && !d.thinking && last && last.role !== 'user';
  }, `Suki's answer to "${text.slice(0, 40)}"`, ms);
}

async function open(url) {
  await m.navigateTo(url);
  await sleep(700);
  return page();
}

async function api(token, method, p, body) {
  const res = await fetch(API + p, {
    method,
    headers: Object.assign({ Accept: 'application/json' }, token ? { Authorization: `Bearer ${token}` } : {}, body !== undefined ? { 'Content-Type': 'application/json' } : {}),
    body: body === undefined ? undefined : (body instanceof Uint8Array ? body : JSON.stringify(body)),
  });
  const text = await res.text();
  let data;
  try {
    data = JSON.parse(text);
  } catch (_) {
    data = text;
  }
  return { status: res.status, data };
}

function sql(query) {
  return execFileSync('docker', ['exec', '-i', PG_CONTAINER, 'psql', '-U', 'syn', '-d', PG_DB, '-At', '-F', '|'], { input: query, encoding: 'utf8' }).trim();
}

// Mail isn't sent locally and codes are stored hashed, so a test sets a code it knows.
function setCode(userId, purpose, code) {
  const hash = crypto.createHash('sha256').update(`${userId}:${code}`).digest('hex');
  sql(`UPDATE email_codes SET code_hash='${hash}', attempts=0 WHERE id=(SELECT id FROM email_codes WHERE user_id='${userId}' AND purpose='${purpose}' AND used_at IS NULL ORDER BY created_at DESC LIMIT 1);`);
}

const PARIS = { key: 'fr-ile-de-france-paris', name: 'Paris, Ile-de-France, France', lat: 48.8566, lon: 2.3522 };

// The server caches city lookups, so a cached answer takes the place of OpenStreetMap.
function seedCity() {
  for (const q of ['paris', 'paris, france']) {
    sql(`INSERT INTO cached_lookups (key, data) VALUES ('city2:${q}', '${JSON.stringify(PARIS)}') ON CONFLICT (key) DO UPDATE SET data=EXCLUDED.data, fetched_at=now();`);
  }
}

const password = () => `T-${crypto.randomBytes(12).toString('hex')}`;

// Another person, through the API only. The simulator can hold one signed-in person at a time.
async function person(name, gender, seeking, birth, extra) {
  const email = `e2e-${name.toLowerCase()}-${Date.now()}@example.com`;
  const pw = password();
  const r = await api('', 'POST', '/v1/auth/signup', { email, password: pw, name, accept_terms: true, language: 'en' });
  if (r.status !== 201) throw new Error(`signup ${name}: ${r.status} ${JSON.stringify(r.data)}`);
  const u = { name, email, password: pw, token: r.data.token, id: r.data.user.id };
  setCode(u.id, 'verify', '123456');
  const v = await api(u.token, 'POST', '/v1/auth/verify', { code: '123456' });
  if (v.status !== 200) throw new Error(`verify ${name}: ${v.status}`);
  return u;
}

function summary() {
  const failed = results.filter((r) => !r.ok);
  console.log(`\n${results.length - failed.length}/${results.length} passed, ${problems.length} console problems`);
  failed.forEach((f) => console.log(`FAILED: ${f.what}`));
  [...new Set(problems)].forEach((p) => console.log(`PROBLEM: ${p}`));
  return failed.length + problems.length;
}

module.exports = {
  API, OUT, sleep, check, note, connect, page, waitUntil, waitPage, waitData, shot, byText, tapText, fill, goto, open, signOut, signIn, sukiSay, api, sql, setCode, password, person, seedCity, PARIS, summary,
  m: () => m,
};
