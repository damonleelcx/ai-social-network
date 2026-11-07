// Notifications, the unread count, pausing, credits, what the buy card says, and voice.
const fs = require('fs');
const path = require('path');

module.exports = async (l) => {
  const m = l.m();
  const account = JSON.parse(fs.readFileSync(path.join(l.OUT, 'main-account.json'), 'utf8'));
  const world = JSON.parse(fs.readFileSync(path.join(l.OUT, 'world.json'), 'utf8'));
  await l.signIn(account);
  const token = await m.callWxMethod('getStorageSync', 'token');

  // Make something unread: Ben writes.
  await l.api(world.ben.token, 'POST', `/v1/matches/${world.match}/messages`, { text: 'Thanks for a good evening!', client_id: `e2e-${Date.now()}` });
  await l.sleep(2000);
  const list = (await l.api(token, 'GET', '/v1/notifications')).data.notifications || [];
  const unread = list.filter((n) => !n.read).length;
  l.note(`Notifications: ${list.length}, unread ${unread}; kinds ${[...new Set(list.map((n) => n.kind))].join(',')}`);

  let p = await l.goto('/pages/me/me');
  let d = await l.waitData(p, (x) => x.user && x.user.id, 'Me', 10000);
  await l.sleep(1000);
  d = await p.data();
  l.check(d.unread === unread, 'Me shows the unread count', `${d.unread} vs ${unread}`);
  await l.shot('me-unread');
  if (unread) l.check(!!(await l.byText(p, '.nav-row', String(unread), 2000).catch(() => null)), 'the count is on the row');

  await l.tapText(p, '.nav-row', 'Notifications');
  p = await l.waitPage('pages/notifications/notifications');
  d = await l.waitData(p, (x) => x.loaded, 'notifications', 10000);
  await l.sleep(500);
  await l.shot('notifications');
  l.check(d.items.filter((x) => x.fresh).length === unread, 'new ones are marked', d.items.filter((x) => x.fresh).length);
  await l.sleep(1000);
  const afterOpen = (await l.api(token, 'GET', '/v1/notifications')).data.notifications || [];
  l.check(afterOpen.every((n) => n.read), 'opening the list reads them');
  const target = d.items.find((x) => x.data && (x.data.screen === 'chat' || x.data.screen === 'proposal' || x.data.screen === 'date'));
  if (target) {
    const rows = await p.$$('.card');
    await rows[d.items.indexOf(target)].tap();
    await l.sleep(1200);
    const c = await l.page();
    const wanted = { chat: 'pages/chat/chat', proposal: 'pages/person/person', date: 'pages/date/date' }[target.data.screen];
    l.check(c.path === wanted, `tapping a ${target.data.screen} notification opens it`, c.path);
    await l.sleep(600);
    await l.shot('notification-target');
    await l.goto('/pages/me/me');
  }
  p = await l.goto('/pages/me/me');
  d = await l.waitData(p, (x) => x.user && x.user.id, 'Me again', 10000);
  await l.sleep(800);
  d = await p.data();
  l.check(d.unread === 0, 'the count clears', d.unread);

  // Pausing.
  l.check(d.canPause === true, 'Me offers a pause', d.canPause);
  const sw = await p.$('.nav-row switch');
  await sw.trigger('change', { value: true });
  await l.sleep(800);
  l.check(l.sql(`SELECT status FROM profiles WHERE user_id='${account.id}';`) === 'paused', 'pausing is saved');
  p = await l.goto('/pages/me/me');
  d = await l.waitData(p, (x) => x.user && x.user.id, 'Me reloaded', 10000);
  await l.sleep(800);
  d = await p.data();
  l.check(d.paused === true, 'pausing survives a reload');
  await l.shot('me-paused');
  await (await p.$('.nav-row switch')).trigger('change', { value: false });
  await l.sleep(800);
  l.check(l.sql(`SELECT status FROM profiles WHERE user_id='${account.id}';`) === 'active', 'unpausing is saved');

  // Credits. No payment runs: the local server has WeChat Pay off.
  d = await p.data();
  l.note(`Credits text: ${d.balanceText} / ${d.creditsText} / packs: ${JSON.stringify(d.packs)}`);
  const credits = (await l.api(token, 'GET', '/v1/credits')).data;
  l.note(`wechat_enabled ${credits.wechat_enabled}, free ${credits.free}, used ${credits.matches_used} of ${credits.free_matches}`);
  l.check(!(credits.matches_used > credits.free_matches && new RegExp(`${credits.matches_used} of`).test(d.creditsText)), 'the credits line never says more used than there are', d.creditsText);
  if (!credits.wechat_enabled) l.check(d.t.noBuying.includes("can't be bought"), 'with WeChat Pay off, Me says credits cannot be bought here');
  await l.shot('me-credits');

  // The buy card, shown by adding it to the page, because a local server with payments off
  // never charges anyone and so never sends it.
  p = await l.goto('/pages/suki/suki');
  d = await l.waitData(p, (x) => x.loaded && x.messages.length > 0, 'Suki', 10000);
  const OUT = "You're out of credits, so I can't answer this one yet.";
  await p.setData({ messages: [{ seq: 999999, role: 'assistant', body: OUT, cards: [{ kind: 'buy_credits' }], replies: [] }], bottom: 'end' });
  await l.sleep(1500);
  await l.shot('buy-card-injected');
  const card = (await p.$$('wx-suki-card'))[0];
  const text = (await card.wxml()).replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ');
  l.note(`Buy card says: ${text}`);
  l.check(/can.t be bought here/.test(text), 'the buy card says credits cannot be bought here', text);
  l.check(!/http|google|stripe|\bapp\b|website/i.test(text), 'the buy card points nowhere else', text);

  // Voice: a real tap on a speaker, and what happens.
  p = await l.goto('/pages/suki/suki');
  d = await l.waitData(p, (x) => x.loaded && x.messages.length > 0, 'Suki', 10000);
  await l.sleep(1500);
  d = await p.data();
  const status = (await l.api(token, 'GET', '/v1/agent/speech')).data;
  l.note(`Server voice enabled: ${status.enabled}; speaker shown: ${d.speaker}`);
  l.check(d.speaker === status.enabled, 'speakers show only when the server has a voice');
  const speakers = await p.$$('.speaker');
  l.note(`${speakers.length} speakers on screen`);
  if (speakers.length) {
    await speakers[speakers.length - 1].tap();
    const phases = [];
    const until = Date.now() + 60000;
    let toast = null;
    while (Date.now() < until) {
      const x = await p.data();
      if (phases[phases.length - 1] !== x.phase) phases.push(x.phase);
      if (x.phase === 'playing') break;
      if (phases.length > 1 && x.phase === null) break;
      await l.sleep(200);
    }
    l.note(`Speaker phases: ${JSON.stringify(phases)}`);
    await l.shot('voice-after-tap');
    l.check(phases.includes('loading'), 'tapping a speaker starts fetching her voice', JSON.stringify(phases));
    // The server needs to reach the voice vendor. If it can't, it answers 502, and the mini
    // program must say so and stop. Either way is reported, not assumed.
    if (phases.includes('playing')) l.check(true, 'the clip starts playing with the real voice');
    else l.note(`THE REAL VOICE DID NOT PLAY: phases ${JSON.stringify(phases)}. Check the server can reach the vendor.`);

    // The player itself, with a stand-in clip: wx.request answers with silent MP3 bytes.
    const frame = Buffer.alloc(417);
    frame[0] = 0xff; frame[1] = 0xfb; frame[2] = 0x90; frame[3] = 0x64;
    const silent = Buffer.concat(Array(60).fill(frame)).toString('base64');
    await m.mockWxMethod('request', function standIn(opts, b64) { // eslint-disable-line
      if (String(opts.url).endsWith('/v1/agent/speech') && opts.method === 'POST') {
        return Promise.resolve({ statusCode: 200, data: wx.base64ToArrayBuffer(b64), header: {}, errMsg: 'request:ok' }); // eslint-disable-line
      }
      return new Promise((resolve, reject) => {
        this.origin(Object.assign({}, opts, { success: resolve, fail: reject }));
      });
    }, silent);
    const seen = [];
    const again = await p.$$('.speaker');
    await again[again.length - 1].tap();
    const stop = Date.now() + 12000;
    while (Date.now() < stop) {
      const x = await p.data();
      if (seen[seen.length - 1] !== x.phase) seen.push(x.phase);
      if (x.phase === 'playing') break;
      if (seen.length > 1 && x.phase === null) break;
      await l.sleep(150);
    }
    await m.restoreWxMethod('request');
    l.note(`With a stand-in clip (request mocked), phases: ${JSON.stringify(seen)}`);
    l.check(seen.includes('playing'), 'the audio player starts a clip in the simulator (bytes mocked; nothing was heard)', JSON.stringify(seen));
    const usage = l.sql(`SELECT coalesce(sum(chars),0) FROM tts_usage WHERE user_id='${account.id}';`);
    l.note(`Voice characters charged today: ${usage}`);
    const written = await m.evaluate(() => new Promise((res) => {
      wx.getFileSystemManager().readdir({ dirPath: `${wx.env.USER_DATA_PATH}/suki-voice`, success: (r) => res(r.files), fail: (e) => res(`ERR ${e.errMsg}`) });
    }));
    l.note(`Clips on the phone: ${JSON.stringify(written)}`);
    await l.sleep(1500);
    if ((await p.data()).phase === 'playing') {
      await (await p.$('.speaker.on')).tap();
      await l.sleep(400);
      l.check((await p.data()).clip === null, 'tapping again stops it');
    }
  }

  // Signing out deletes the clips.
  await l.signOut();
  const left = await m.evaluate(() => new Promise((res) => {
    wx.getFileSystemManager().readdir({ dirPath: `${wx.env.USER_DATA_PATH}/suki-voice`, success: (r) => res(r.files.length), fail: () => res(0) });
  }));
  l.check(left === 0, 'signing out deletes her clips', left);
};
