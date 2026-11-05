// Her face where two people are involved: the introduction's pitch, the person page, the
// empty chat and the coaching panel. Two new people, with real photo review and a real
// search. Screenshots are taken only after checking the page, and this flow never opens a
// real photo preview, which would sit over later screenshots.
const fs = require('fs');
const path = require('path');

const SUKI = /assets\/suki\/(\w+)\.jpg/;

function photoFiles() {
  if (!process.env.WX_PHOTOS) throw new Error('Set WX_PHOTOS to a folder of test photos with faces');
  const out = [];
  const walk = (d) => fs.readdirSync(d, { withFileTypes: true }).forEach((e) => {
    const p = path.join(d, e.name);
    if (e.isDirectory()) walk(p);
    else if (/\.jpe?g$/i.test(e.name)) out.push(p);
  });
  walk(process.env.WX_PHOTOS);
  return out;
}

module.exports = async (l) => {
  const m = l.m();
  const shot = async (name, wanted, check) => {
    const c = await l.page();
    l.check(c.path === wanted, `the ${name} screenshot is of ${wanted}`, c.path);
    if (check) l.check(await check(await c.data()), `the ${name} screenshot has what it should`);
    return l.shot(name);
  };
  l.seedCity();
  const files = photoFiles();
  const make = async (name, gender, seeking, i) => {
    const u = await l.person(name);
    const s = await l.api(u.token, 'POST', '/v1/profile', { birthdate: '1993-03-03', gender, seeking: [seeking], goal: 'long_term', interests: ['jazz', 'hiking', 'ramen'], bio: 'Weekend hiker who plays jazz records.', age_min: 25, age_max: 45, smoking: 'no' });
    if (s.status !== 200) throw new Error(`${name}: ${s.status} ${JSON.stringify(s.data)}`);
    await l.api(u.token, 'POST', '/v1/profile/city', { query: 'Paris' });
    const up = await l.api(u.token, 'POST', '/v1/photos', new Uint8Array(fs.readFileSync(files[i % files.length])));
    if (up.status !== 201) throw new Error(`${name} photo: ${up.status}`);
    return u;
  };
  const hana = await make('Hana', 'woman', 'man', 0);
  const kai = await make('Kai', 'man', 'woman', 1);

  // A real search between them. It may need asking for, depending on who went live first.
  const proposal = await l.waitUntil(async () => {
    const list = (await l.api(hana.token, 'GET', '/v1/proposals')).data.proposals || [];
    return list.find((x) => x.person.display_name === 'Kai');
  }, 'a real introduction', 150000);
  l.note(`Pitch: ${proposal.pitch}`);

  await l.signIn(hana);
  let p = await l.goto('/pages/matches/matches');
  await l.waitData(p, (d) => d.loaded && d.proposals.length > 0, 'the introduction', 15000);
  await l.sleep(1500);
  const cards = await p.$$('wx-suki-card');
  const card = await cards[0].wxml();
  l.check(/suki-avatar/.test(card) && SUKI.exec(card) && SUKI.exec(card)[1] === 'playful', 'the introduction on Matches has her beside the pitch', card.slice(0, 200));
  await shot('matches-intro', 'pages/matches/matches');

  p = await l.goto(`/pages/person/person?proposal=${proposal.id}`);
  await l.waitData(p, (d) => d.person, 'the person page', 15000);
  await l.sleep(1200);
  await shot('person-pitch', 'pages/person/person', (d) => !!d.proposal.pitch);

  // Both say yes.
  await l.api(hana.token, 'POST', `/v1/proposals/${proposal.id}/decide`, { decision: 'accept' });
  const made = await l.api(kai.token, 'POST', `/v1/proposals/${proposal.id}/decide`, { decision: 'accept' });
  const match = made.data.match_id;
  l.check(!!match, 'a real match', JSON.stringify(made.data));

  // The chat. She leaves a real note when a match starts.
  p = await l.goto(`/pages/chat/chat?id=${match}`);
  await l.waitData(p, (d) => d.match && d.match.status === 'active' && d.loaded, 'the chat', 15000);
  await l.sleep(1000);
  const start = await p.data();
  l.note(`Messages at the start: ${start.messages.length}; hint yet: ${!!start.hint}`);
  if (!start.messages.length) await shot('chat-empty', 'pages/chat/chat', (d) => !!d.sayHi);
  let d = await l.waitData(p, (x) => x.hint, 'her first hint', 90000).catch(() => null);
  if (!d) {
    await (await p.$('.ask-row .chip')).tap();
    await l.waitData(p, (x) => x.asking, 'asking', 4000);
    await l.sleep(600);
    await shot('chat-asking', 'pages/chat/chat', (x) => x.asking);
    d = await l.waitData(p, (x) => x.hint, 'a hint after asking', 120000);
  }
  await l.sleep(800);
  const coach = await p.$('.coach');
  l.check(!!coach, 'the coaching panel is on the page');
  const panel = await coach.wxml();
  l.check(SUKI.exec(panel) && SUKI.exec(panel)[1] === 'playful', 'the panel has her playful face', panel.slice(0, 200));
  l.check(panel.includes(d.hint.body.slice(0, 20)), 'the panel shows the hint from the server');
  await shot('chat-hint', 'pages/chat/chat', (x) => !!x.hint);
  l.note(`Hint on screen: ${d.hint.body}`);

  // Asking again, with a note up: the chip shows it's working.
  await (await p.$('.ask-row .chip')).tap();
  await l.sleep(500);
  await shot('chat-hint-asking', 'pages/chat/chat', (x) => x.asking);

  // With the keyboard up the panel steps aside to one line, face included.
  await (await p.$('.composer input')).trigger('focus', {});
  await l.sleep(500);
  const line = await p.$('.coach-line');
  l.check(!!line && SUKI.test(await line.wxml()), 'with the keyboard up, one line with her face');
  await shot('chat-hint-typing', 'pages/chat/chat', (x) => x.typing);
  await (await p.$('.composer input')).trigger('blur', {});
  clearTimeout(0);
};
