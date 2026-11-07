// Three more people in the same city, made through the API with real photo review and real
// searches. The main account decides in the UI; the others answer through the API.
const fs = require('fs');
const path = require('path');

const PHOTOS = process.env.WX_PHOTOS;
const SCORE = /\b(score|rank|percent|prior|compat(ibility)?)\b|\d+\s*%/i;

function photoFiles() {
  if (!PHOTOS) throw new Error('Set WX_PHOTOS to a folder of test photos with faces');
  const out = [];
  const walk = (d) => fs.readdirSync(d, { withFileTypes: true }).forEach((e) => {
    const p = path.join(d, e.name);
    if (e.isDirectory()) walk(p);
    else if (/\.jpe?g$/i.test(e.name)) out.push(p);
  });
  walk(PHOTOS);
  return out;
}

module.exports = async (l) => {
  const m = l.m();
  const account = JSON.parse(fs.readFileSync(path.join(l.OUT, 'main-account.json'), 'utf8'));
  await l.signIn(account);
  const token = await m.callWxMethod('getStorageSync', 'token');
  const worldFile = path.join(l.OUT, 'world.json');
  let world = fs.existsSync(worldFile) ? JSON.parse(fs.readFileSync(worldFile, 'utf8')) : null;

  let p = await l.goto('/pages/suki/suki');
  await l.waitData(p, (d) => d.loaded, 'Suki loaded');

  if (!world) {
    // Nobody else exists yet, so a search she's asked for should say so and keep talking.
    await l.sukiSay(p, 'Please look for someone for me right now.');
    const d = await l.waitData(p, (x) => x.messages.some((n) => /Nobody new near you|keep me company/i.test(n.body || '')), 'the nobody-here line', 180000).catch((e) => ({ err: e.message }));
    l.check(!d.err, 'with nobody around, Suki says so and invites a chat', d.err);
    await l.shot('suki-nobody');

    const files = photoFiles();
    const make = async (name, birth, goal, interests, bio, i) => {
      const u = await l.person(name);
      const s = await l.api(u.token, 'POST', '/v1/profile', { birthdate: birth, gender: 'man', seeking: ['woman'], goal, interests, bio, age_min: 25, age_max: 40, occupation: 'Teacher', smoking: 'no', drinking: 'sometimes', prompts: [{ q: 'My ideal Sunday', a: `${name} goes for a long walk.` }] });
      if (s.status !== 200) throw new Error(`${name} profile: ${s.status} ${JSON.stringify(s.data)}`);
      await l.api(u.token, 'POST', '/v1/profile/city', { query: 'Paris' });
      const up = await l.api(u.token, 'POST', '/v1/photos', new Uint8Array(fs.readFileSync(files[(i + 1) % files.length])));
      if (up.status !== 201) throw new Error(`${name} photo: ${up.status}`);
      return u;
    };
    world = {
      ben: await make('Ben', '1992-03-03', 'long_term', ['jazz', 'hiking', 'ramen'], 'Weekend hiker who plays jazz records.', 0),
      cal: await make('Cal', '1990-05-05', 'long_term', ['jazz', 'climbing', 'coffee'], 'I climb and I drink too much coffee.', 1),
      dan: await make('Dan', '1988-07-07', 'casual', ['poker', 'cars', 'gym'], 'Here for fun, nothing serious.', 2),
    };
    fs.writeFileSync(worldFile, JSON.stringify(world));
  }

  // Their photos go through the real review, and their profiles go live.
  for (const k of ['ben', 'cal', 'dan']) {
    const st = await l.waitUntil(async () => {
      const ph = (await l.api(world[k].token, 'GET', '/v1/photos')).data.photos || [];
      return ph[0] && ph[0].status !== 'pending' ? ph[0] : null;
    }, `${k}'s photo review`, 120000).catch((e) => ({ status: e.message }));
    l.note(`${k}: real photo review said ${st.status} ${st.reason || ''}`);
    if (st.status !== 'approved') {
      l.note(`APPROVED IN THE DATABASE ONLY TO CONTINUE: ${k}`);
      l.sql(`UPDATE photos SET status='approved' WHERE user_id='${world[k].id}';`);
    }
    const prof = (await l.api(world[k].token, 'GET', '/v1/profile')).data;
    l.note(`${k}: status ${prof.profile.status}, missing ${JSON.stringify(prof.missing)}`);
  }

  // Ask for a search again now people exist. It may already have been done by their own.
  let proposals = (await l.api(token, 'GET', '/v1/proposals')).data.proposals || [];
  if (!proposals.length) {
    await l.sukiSay(p, 'Is there anyone new for me now? Please search again.');
    await l.waitUntil(async () => {
      proposals = (await l.api(token, 'GET', '/v1/proposals')).data.proposals || [];
      return proposals.length > 0;
    }, 'a real introduction', 240000).catch((e) => l.check(false, 'a real search introduces someone', e.message));
  }
  l.note(`Ana has ${proposals.length} proposals: ${proposals.map((x) => `${x.person.display_name}/${x.status}`).join(', ')}`);
  l.note(`Pitches: ${JSON.stringify(proposals.map((x) => x.pitch))}`);
  l.check(proposals.length > 0, 'a real search introduces someone');
  await l.sleep(1000);
  await l.shot('suki-with-intro');

  // Matches tab: the card.
  p = await l.goto('/pages/matches/matches');
  await l.waitData(p, (d) => d.loaded, 'Matches loaded', 20000);
  await l.sleep(1500);
  await l.shot('matches');
  const cards = await p.$$('wx-suki-card');
  l.check(cards.length === proposals.filter((x) => x.status === 'open' && !x.my_decision).length, 'Matches shows the open introductions', cards.length);
  const allText = JSON.stringify(proposals);
  l.check(!SCORE.test(allText), 'proposals carry no score or rank', allText.match(SCORE));
  if (!cards.length) return;

  // Person page from the card, photo preview, then decide.
  const cardFor = async (list, name) => {
    for (const c of list) if ((await c.wxml()).includes(name)) return c;
    return null;
  };
  const link = await (await cardFor(cards, 'Ben') || cards[0]).$('.grow');
  await link.tap();
  p = await l.waitPage('pages/person/person');
  let d = await l.waitData(p, (x) => x.person, 'person page', 15000);
  await l.sleep(1200);
  await l.shot('person-top');
  l.check(!SCORE.test(JSON.stringify(d.details) + d.line + JSON.stringify(d.person)), 'the person page shows no score or rank');
  l.check(d.open === true, 'the person page offers a decision');
  await m.mockWxMethod('previewImage', function capture(opts) { getApp().lastPreview = opts; }); // eslint-disable-line
  const ph = await l.waitUntil(() => p.$('.hero-photo'), 'a photo on the person page').catch(() => null);
  l.check(!!ph, 'the person page shows a photo');
  if (ph) {
    await ph.tap();
    await l.sleep(1200);
    const preview = await m.evaluate(() => getApp().lastPreview);
    l.check(preview && preview.urls && preview.urls.length >= 1 && !!preview.current, 'tapping a photo asks for the whole photo (previewImage mocked to record it)', JSON.stringify(preview));
    if (preview) l.note(`preview urls are local temp files: ${String(preview.current).slice(0, 40)}`);
  }
  await m.restoreWxMethod('previewImage');

  const firstName = d.person.display_name;
  const firstId = d.proposal.id;
  world.first = { name: firstName, proposal: firstId };
  await l.tapText(p, '.bar button', d.t.yes);
  await l.sleep(2500);
  d = await p.data();
  l.check(!!d.waiting || !!d.chatId, `accepting ${firstName} shows the waiting line or the chat`, JSON.stringify([d.waiting, d.chatId]));
  await l.shot('person-accepted');

  // The other side says yes, and it's a match. Their proposal comes from the same row.
  const other = Object.values(world).find((u) => u.name === firstName);
  const yes = await l.api(other.token, 'POST', `/v1/proposals/${firstId}/decide`, { decision: 'accept' });
  l.check(yes.status === 200 && !!yes.data.match_id, `${firstName} saying yes makes a match`, JSON.stringify(yes.data));
  world.match = yes.data.match_id;
  world.partner = other.name.toLowerCase();

  // Pass on another, from the card in Matches.
  p = await l.goto('/pages/matches/matches');
  await l.waitData(p, (x) => x.loaded && x.matches.length > 0, 'the match on Matches', 20000).catch((e) => l.check(false, 'the match shows on Matches', e.message));
  await l.sleep(1200);
  await l.shot('matches-with-match');
  const left = await p.$$('wx-suki-card');
  if (left.length) {
    const target = await cardFor(left, 'Dan') || left[0];
    const btn = await l.waitUntil(() => target.$('button.plain'), 'pass button');
    await btn.tap();
    await l.sleep(1500);
    await l.shot('matches-passed');
    const after = (await l.api(token, 'GET', '/v1/proposals')).data.proposals || [];
    l.check(after.some((x) => x.my_decision === 'pass') || after.length < proposals.length, 'passing from the card is recorded', JSON.stringify(after.map((x) => [x.person.display_name, x.status, x.my_decision])));
  } else {
    l.note('No second introduction to pass on.');
  }
  fs.writeFileSync(worldFile, JSON.stringify(world));

  // Evidence that decisions become hidden taste signals. Read only.
  await l.sleep(3000);
  const signals = l.sql(`SELECT kind, label, strength, features FROM hidden.signals WHERE user_id='${account.id}' ORDER BY id;`);
  l.note(`Ana's taste signals:\n${signals}`);
  l.check(signals.length > 0, 'accepting and passing records taste signals');
};
