// Suki is visible everywhere she should be, in the running app. Every screenshot is taken
// after checking the current page, and with no photo preview open. Needs no earlier flow.
const SUKI = /assets\/suki\/(\w+)\.jpg/;

// Reads the image inside each avatar, as the page really has it.
async function faces(p) {
  const out = [];
  for (const a of await p.$$('wx-suki-avatar')) {
    const m = SUKI.exec(await a.wxml());
    out.push(m ? m[1] : null);
  }
  // An avatar inside another component can't be selected, but it's in that component's markup.
  for (const a of await p.$$('wx-suki-says')) {
    const m = SUKI.exec(await a.wxml());
    out.push(m ? m[1] : null);
  }
  return out;
}

module.exports = async (l) => {
  const m = l.m();
  const shot = async (name, wanted) => {
    const c = await l.page();
    l.check(c.path === wanted, `the ${name} screenshot is of ${wanted}`, c.path);
    return l.shot(name);
  };

  // Login, before anyone is signed in.
  await l.signOut();
  let p = await l.goto('/pages/login/login');
  await l.waitUntil(() => p.$('wx-suki-avatar'), 'the login avatar');
  await m.mockWxMethod('showActionSheet', { tapIndex: 1, errMsg: 'showActionSheet:ok' });
  await (await p.$('.hero .link')).tap();
  await l.sleep(400);
  await m.restoreWxMethod('showActionSheet');
  await l.sleep(500);
  l.check((await faces(p))[0] === 'portrait', 'login shows her portrait', (await faces(p))[0]);
  await shot('login-wechat', 'pages/login/login');
  await l.tapText(p, '.link', 'Sign in with email');
  await l.sleep(500);
  l.check((await faces(p))[0] === 'playful', 'email sign-in shows her playful face', (await faces(p))[0]);
  await shot('login-email', 'pages/login/login');

  // A new account: no photo, so she stands in on Me.
  const u = await l.person('Mia');
  await l.signIn(u);
  p = await l.goto('/pages/me/me');
  await l.waitData(p, (d) => d.user && d.user.id, 'Me', 10000);
  await l.sleep(1500);
  const me = await faces(p);
  l.check(me[0] === 'pouty', 'Me with no photo shows the pouty face', JSON.stringify(me));
  l.check((await p.data()).t.noPhoto.includes('my face'), 'Me says it is her face, not yours');
  await shot('me-no-photo', 'pages/me/me');

  // Her first message and a real answer. Moods come from the real model, so two moods are
  // also set by hand on real messages' copies, and said so.
  p = await l.goto('/pages/suki/suki');
  await l.waitData(p, (d) => d.loaded && d.messages.length > 0, 'her greeting', 20000);
  await l.sleep(1000);
  await shot('suki-greeting', 'pages/suki/suki');
  await l.fill(p, '.composer input', "Hi! I'm Mia. Tell me a joke about first dates.");
  await (await p.$('.composer .send')).tap();
  await l.waitData(p, (d) => d.thinking, 'her thinking', 8000).catch(() => null);
  await l.sleep(300);
  const busy = await p.data();
  if (busy.thinking) {
    l.check(!!(await p.$('.suki-typing wx-suki-avatar')), 'the typing state shows her thinking face');
    await shot('suki-typing', 'pages/suki/suki');
  } else {
    l.note('She answered before the typing state could be seen.');
  }
  await l.waitData(p, (d) => {
    const last = d.messages[d.messages.length - 1];
    return !d.thinking && last && last.role === 'assistant';
  }, 'her answer', 120000);
  await l.sleep(800);

  // Every message from her has an avatar with the right face.
  const data = await p.data();
  const hers = data.messages.filter((x) => x.role !== 'user' && x.role !== 'event');
  const rows = await p.$$('.suki-row');
  const ringed = [];
  for (const r of rows) {
    const face = SUKI.exec(await r.wxml());
    ringed.push(face ? face[1] : null);
  }
  l.check(ringed.length === hers.length, 'one row per message from her', `${ringed.length} rows, ${hers.length} messages`);
  l.check(ringed.every(Boolean), 'every message from her has her face', JSON.stringify(ringed));
  hers.forEach((x, i) => {
    const want = ['happy', 'surprised', 'thinking', 'playful', 'pouty', 'portrait', 'chibi'].includes(x.mood) ? x.mood : 'portrait';
    l.check(ringed[i] === want, `message ${x.seq} (mood "${x.mood}") shows ${want}`, ringed[i]);
  });
  l.note(`Real moods from the model: ${JSON.stringify(hers.map((x) => x.mood))}`);
  await shot('suki-real-answer', 'pages/suki/suki');

  // Two moods side by side, whatever the model chose: real rows, moods set in the page's data.
  const set = {};
  hers.slice(-2).forEach((x, i) => {
    set[`messages[${data.messages.indexOf(x)}].mood`] = i ? 'playful' : 'pouty';
  });
  if (hers.length < 2) l.note('Only one message from her to show moods on.');
  await p.setData(set);
  await l.sleep(600);
  const mixed = await faces(p);
  l.check(hers.length < 2 || (mixed.includes('pouty') && mixed.includes('playful')), 'two moods show as two faces (moods set by hand)', JSON.stringify(mixed));
  await shot('suki-two-moods', 'pages/suki/suki');

  // The setting: off means her portrait everywhere in the conversation.
  const token = await m.callWxMethod('getStorageSync', 'token');
  p = await l.goto('/pages/settings/suki/suki');
  let d = await l.waitData(p, (x) => x.s, 'her settings', 10000);
  await l.sleep(600);
  l.check(d.s.mood_avatar === true, 'Avatar moods starts on');
  const settingsFaces = await faces(p);
  l.check(settingsFaces[0] === d.faces[d.s.suki_tone], 'her settings show her face for the tone', JSON.stringify(settingsFaces));
  await shot('suki-settings', 'pages/settings/suki/suki');
  await (await p.$('switch[data-f=mood_avatar]')).trigger('change', { value: false });
  await l.sleep(600);
  l.check((await l.api(token, 'GET', '/v1/settings')).data.settings.mood_avatar === false, 'Avatar moods saves');
  p = await l.goto('/pages/suki/suki');
  await l.waitData(p, (x) => x.loaded && x.messages.length > 0 && x.moodAvatar === false, 'moods off', 10000);
  await p.setData({ [`messages[${(await p.data()).messages.length - 1}].mood`]: 'playful' });
  await l.sleep(600);
  const off = await faces(p);
  l.check(off.every((f) => f === 'portrait'), 'with moods off, every avatar is her portrait', JSON.stringify(off));
  await shot('suki-moods-off', 'pages/suki/suki');
  await l.api(token, 'POST', '/v1/settings', { mood_avatar: true });

  // Her empty states.
  for (const [url, face] of [['/pages/matches/matches', 'thinking'], ['/pages/dates/dates', 'playful'], ['/pages/notifications/notifications', 'happy'], ['/pages/settings/blocked/blocked', 'happy'], ['/pages/settings/memory/memory', 'thinking']]) {
    p = await l.goto(url);
    await l.sleep(1800);
    const list = await faces(p);
    l.note(`${url}: faces ${JSON.stringify(list)}`);
    const name = url.split('/').pop();
    if (name === 'notifications' && (await p.data()).items.length) continue;
    l.check(list.includes(face), `${name} shows ${face} when empty`, JSON.stringify(list));
    await shot(`empty-${name}`, url.slice(1));
  }

  // The tab bar is shown on a tab page, so this is what the last tab screenshots show too.
  p = await l.goto('/pages/matches/matches');
  await l.sleep(1200);
  await shot('tabs-on-matches', 'pages/matches/matches');
};
