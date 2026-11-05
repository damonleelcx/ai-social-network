// Every setting saves through POST and survives a reload. wx.request is wrapped to record
// the methods sent. Then language, memory and blocked people.
const fs = require('fs');
const path = require('path');

const recordRequests = (m) => m.mockWxMethod('request', function spy(opts) { // eslint-disable-line
  const app = getApp(); // eslint-disable-line
  app.sent = app.sent || [];
  app.sent.push(`${opts.method || 'GET'} ${String(opts.url).replace(/^https?:\/\/[^/]+/, '')}`);
  // A mock answers by resolving, not by calling the callbacks itself.
  return new Promise((resolve, reject) => {
    this.origin(Object.assign({}, opts, { success: resolve, fail: reject }));
  });
});

const APP_SWITCHES = ['notify_matches', 'notify_messages', 'notify_dates', 'notify_suki', 'notify_email', 'read_receipts'];
const SUKI_SWITCHES = ['suki_voice', 'suki_voice_auto', 'chat_coach', 'chat_coach_read', 'chat_prompts', 'memory_enabled', 'debrief_enabled'];
const SUKI_CHOICES = { suki_tone: 'direct', suki_emoji: 'lots', suki_proactive: 'low' };

async function toggles(l, url, names, token) {
  let p = await l.goto(url);
  let d = await l.waitData(p, (x) => x.s, 'settings', 10000);
  const start = d.s;
  for (const f of names) {
    const el = await l.waitUntil(() => p.$(`switch[data-f=${f}]`), `switch ${f}`, 5000).catch(() => null);
    if (!el) {
      l.check(false, `${f} has a switch`);
      continue;
    }
    if (f === 'suki_voice_auto' && !start.suki_voice) continue;
    const next = !start[f];
    await el.trigger('change', { value: next });
    await l.sleep(500);
    const server = (await l.api(token, 'GET', '/v1/settings')).data.settings[f];
    l.check(server === next, `${f} saves`, `server has ${server}`);
  }
  await l.shot(`settings-${names[0]}`);
  p = await l.goto(url);
  d = await l.waitData(p, (x) => x.s, 'settings after reload', 10000);
  for (const f of names) l.check(d.s[f] === !start[f], `${f} survives a reload`, d.s[f]);
  // Put them back.
  const back = {};
  names.forEach((f) => { back[f] = start[f]; });
  await l.api(token, 'POST', '/v1/settings', back);
  return p;
}

module.exports = async (l) => {
  const m = l.m();
  const account = JSON.parse(fs.readFileSync(path.join(l.OUT, 'main-account.json'), 'utf8'));
  const world = JSON.parse(fs.readFileSync(path.join(l.OUT, 'world.json'), 'utf8'));
  await l.signIn(account);
  const token = await m.callWxMethod('getStorageSync', 'token');

  await recordRequests(m);
  let p = await toggles(l, '/pages/settings/app/app', APP_SWITCHES, token);
  await l.tapText(p, '.chip', '5');
  await l.sleep(500);
  l.check((await l.api(token, 'GET', '/v1/settings')).data.settings.daily_matches === 5, 'introductions per day saves');

  p = await toggles(l, '/pages/settings/suki/suki', SUKI_SWITCHES, token);
  await l.waitData(p, (x) => x.s && x.t, 'Suki settings');
  await l.sleep(400);
  await l.shot('settings-suki');
  for (const [f, v] of Object.entries(SUKI_CHOICES)) {
    await (await p.$(`.chip[data-f=${f}][data-v=${v}]`)).tap();
    await l.sleep(500);
    l.check((await l.api(token, 'GET', '/v1/settings')).data.settings[f] === v, `${f} saves`);
  }
  const nick = await p.$('input[maxlength="30"]');
  await nick.input('Annie');
  await nick.trigger('blur', { value: 'Annie' });
  await l.sleep(500);
  l.check((await l.api(token, 'GET', '/v1/settings')).data.settings.suki_nickname === 'Annie', 'her name for me saves');
  p = await l.goto('/pages/settings/suki/suki');
  const reloaded = await l.waitData(p, (x) => x.s, 'Suki settings again', 10000);
  l.check(reloaded.s.suki_tone === 'direct' && reloaded.s.suki_nickname === 'Annie', 'Suki settings survive a reload');
  await l.api(token, 'POST', '/v1/settings', { suki_tone: 'warm', suki_emoji: 'some', suki_proactive: 'normal', suki_nickname: '' });

  const sent = await m.evaluate(() => getApp().sent || []);
  await m.restoreWxMethod('request');
  const saves = sent.filter((s) => s.endsWith('/v1/settings') && !s.startsWith('GET'));
  l.note(`Setting saves seen by wx.request: ${saves.length}, methods: ${[...new Set(saves.map((s) => s.split(' ')[0]))].join(',')}`);
  l.check(saves.length > 0 && saves.every((s) => s.startsWith('POST')), 'settings save with POST, never PATCH', JSON.stringify(saves.slice(0, 3)));
  l.check(!sent.some((s) => s.startsWith('PATCH')), 'no PATCH anywhere in this run');

  // Language after sign-in: applies to the tab bar and pages, and Suki's setting.
  p = await l.goto('/pages/settings/app/app');
  await l.waitData(p, (x) => x.s, 'app settings');
  await (await p.$('.chip[data-v=ja]')).tap();
  await l.waitData(p, (x) => x.s.language === 'ja', 'Japanese saved', 5000);
  await l.sleep(600);
  await l.shot('settings-ja');
  l.check((await l.api(token, 'GET', '/v1/settings')).data.settings.language === 'ja', 'language saves on the account');
  const q = await l.goto('/pages/me/me');
  await l.waitData(q, (x) => x.user && x.user.id, 'Me in Japanese');
  await l.sleep(700);
  await l.shot('me-ja');
  const ascii = (await q.data()).t;
  const untranslated = Object.values(ascii).filter((v) => /^[\x20-\x7e]+$/.test(v) && /[a-z]{3}/i.test(v) && !/Suki|SynAgent/.test(v));
  l.check(untranslated.length === 0, 'Me has no untranslated words in Japanese', untranslated.join(' | '));
  p = await l.goto('/pages/settings/app/app');
  await l.waitData(p, (x) => x.s, 'app settings');
  await (await p.$('.chip[data-v=en]')).tap();
  await l.waitData(p, (x) => x.s.language === 'en', 'English saved', 5000);

  // Memory.
  // Her real conversation may not have left a note, so one is added for the delete to act on.
  l.sql(`INSERT INTO memories (user_id, content) VALUES ('${account.id}', 'E2E note: likes jazz');`);
  p = await l.goto('/pages/settings/memory/memory');
  let d = await l.waitData(p, (x) => x.items !== null, 'the memory screen', 10000);
  await l.sleep(400);
  await l.shot('memory');
  l.note(`Suki's notes: ${d.items.map((n) => n.note).join(' | ')}`);
  if (d.items.length) {
    await (await p.$('.chip.remove')).tap();
    await l.sleep(600);
    const left = (await l.api(token, 'GET', '/v1/memories')).data.memories || [];
    l.check(left.length === d.items.length - 1, 'deleting a note removes it', left.length);
  } else {
    l.note('No notes to delete, so that step did not run.');
  }
  await m.mockWxMethod('showModal', { confirm: true, cancel: false, errMsg: 'showModal:ok' });
  await l.tapText(p, 'button', 'Forget everything');
  await l.sleep(1500);
  await m.restoreWxMethod('showModal');
  d = await p.data();
  l.check(d.items.length === 0, 'starting over empties the list', d.items.length);
  await l.shot('memory-empty');

  // Blocked people.
  p = await l.goto('/pages/settings/blocked/blocked');
  d = await l.waitData(p, (x) => x.items !== null, 'blocked people', 10000);
  await l.sleep(300);
  await l.shot('blocked');
  l.check(d.items.some((b) => b.user_id === world.cal.id), 'the person I blocked is listed', JSON.stringify(d.items));
  if (d.items.length) {
    await m.mockWxMethod('showModal', { confirm: true, cancel: false, errMsg: 'showModal:ok' });
    await (await p.$('.chip.unblock')).tap();
    await l.sleep(1200);
    await m.restoreWxMethod('showModal');
    d = await p.data();
    l.check(!d.items.some((b) => b.user_id === world.cal.id), 'unblocking takes them off the list', JSON.stringify(d.items));
    await l.shot('blocked-empty');
  }
};
