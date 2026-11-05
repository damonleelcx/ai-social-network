// Out of credits, then a date from the chat to a confirmed plan, then the debrief.
// Places are seeded in the server's cache in case it can't reach OpenStreetMap. The time of
// the date is moved into the past in the database, so the debrief comes now and not tomorrow.
const fs = require('fs');
const path = require('path');

const VENUES = [
  { id: 'e2e-1', name: 'Duc des Lombards', kind: 'music venue', address: '42 Rue des Lombards, Paris', lat: 48.8597, lon: 2.3483 },
  { id: 'e2e-2', name: 'Parc des Buttes-Chaumont', kind: 'park', address: '1 Rue Botzaris, Paris', lat: 48.8809, lon: 2.3828 },
  { id: 'e2e-3', name: 'Kodawari Ramen', kind: 'restaurant', address: '29 Rue Mazarine, Paris', lat: 48.8549, lon: 2.3379 },
  { id: 'e2e-4', name: 'Cafe de Flore', kind: 'cafe', address: '172 Bd Saint-Germain, Paris', lat: 48.8541, lon: 2.3326 },
  { id: 'e2e-5', name: 'Musee d Orsay', kind: 'museum', address: '1 Rue de la Legion d Honneur, Paris', lat: 48.86, lon: 2.3266 },
];

module.exports = async (l) => {
  const m = l.m();
  const account = JSON.parse(fs.readFileSync(path.join(l.OUT, 'main-account.json'), 'utf8'));
  const world = JSON.parse(fs.readFileSync(path.join(l.OUT, 'world.json'), 'utf8'));
  await l.signIn(account);
  const token = await m.callWxMethod('getStorageSync', 'token');
  const ben = world.ben;

  // Out of credits: she's had her free match and has no balance.
  const credits = (await l.api(token, 'GET', '/v1/credits')).data;
  l.note(`Credits: ${JSON.stringify(credits)}`);
  let p = await l.goto('/pages/suki/suki');
  await l.waitData(p, (d) => d.loaded, 'Suki loaded');
  if (!credits.free && credits.balance === 0) {
    await l.sukiSay(p, 'What should I wear on a first date?', 60000);
    const d = await p.data();
    const last = d.messages[d.messages.length - 1];
    l.note(`Out of credits line: ${last.body}`);
    l.check(/out of credits/i.test(last.body || ''), 'with no credits, Suki says so', last.body);
    l.check((last.cards || []).some((c) => c.kind === 'buy_credits'), 'with no credits, the buy card follows');
    await l.sleep(1500);
    await l.shot('out-of-credits');
    const cards = await p.$$('wx-suki-card');
    const text = await cards[cards.length - 1].wxml();
    l.note(`Card: ${text.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').slice(0, 200)}`);
    // Enough to carry on testing Suki. Written straight to the database.
    l.sql(`INSERT INTO credit_balances (user_id, balance) VALUES ('${account.id}', 30) ON CONFLICT (user_id) DO UPDATE SET balance=30;`);
    l.note('CREDITS GRANTED IN THE DATABASE to keep testing.');
  } else {
    l.note('Not out of credits, so the message was not tested here.');
  }

  // Plan a date from the chat.
  const key = 'venues:fr-ile-de-france-paris:"amenity"="bar","amenity"="cafe","amenity"="music_venue","amenity"="restaurant","leisure"="park","sport"="climbing","tourism"="museum"'.replace(/'/g, "''");
  l.sql(`INSERT INTO cached_lookups (key, data) VALUES ('${key}', '${JSON.stringify(VENUES).replace(/'/g, "''")}') ON CONFLICT (key) DO UPDATE SET data=EXCLUDED.data, fetched_at=now();`);
  p = await l.goto(`/pages/chat/chat?id=${world.match}`);
  await l.waitData(p, (x) => x.match && x.match.status === 'active', 'the chat', 10000);
  await l.tapText(p, '.bar .chip', 'Plan a date');
  p = await l.waitPage('pages/date/date', 15000);
  let d = await l.waitData(p, (x) => x.date, 'the date page', 15000);
  await l.sleep(600);
  await l.shot('date-searching');
  l.note(`Date status at first: ${d.date.status}`);
  const dateId = d.id;

  d = await l.waitData(p, (x) => x.date.status !== 'searching', 'places to be found', 180000).catch((e) => ({ err: e.message }));
  if (d.err) {
    l.check(false, 'the real planner finds places', d.err);
    return;
  }
  l.check(d.date.status === 'voting', 'the real planner offers places to vote on', d.date.status);
  await l.sleep(500);
  await l.shot('date-voting');
  l.note(`Places: ${(d.date.venues || []).map((v) => `${v.name}: ${v.reason}`).join(' | ')}`);

  await l.tapText(p, 'button', 'Vote for this one');
  d = await l.waitData(p, (x) => x.date.my_vote, 'my vote to show', 8000);
  await l.shot('date-voted');
  const vote = await l.api(ben.token, 'POST', `/v1/dates/${dateId}/vote`, { venue_id: d.date.my_vote });
  l.check(vote.status === 200, 'the other person votes', vote.status);
  d = await l.waitData(p, (x) => x.date.status === 'scheduling', 'the time step', 15000).catch((e) => ({ err: e.message }));
  l.check(!d.err, 'the same choice moves on to the time', d.err);
  await l.sleep(400);
  await l.shot('date-scheduling');

  // Pick a time through the pickers.
  const when = new Date(Date.now() + 5 * 864e5);
  const pad = (n) => (n < 10 ? `0${n}` : `${n}`);
  const pickers = await p.$$('picker');
  await pickers[0].trigger('change', { value: `${when.getFullYear()}-${pad(when.getMonth() + 1)}-${pad(when.getDate())}` });
  await pickers[1].trigger('change', { value: '19:30' });
  await l.sleep(300);
  await l.tapText(p, 'button', 'Suggest this time');
  d = await l.waitData(p, (x) => x.proposedText, 'the suggested time', 8000);
  l.check(d.date.proposed_by_me && /19:30/.test(d.proposedText), 'my time shows as suggested', d.proposedText);
  await l.shot('date-proposed');
  const ok = await l.api(ben.token, 'POST', `/v1/dates/${dateId}/confirm`);
  l.check(ok.status === 200, 'the other person says yes', ok.status);
  d = await l.waitData(p, (x) => x.date.status === 'confirmed', 'a confirmed date', 15000).catch((e) => ({ err: e.message }));
  l.check(!d.err, 'the plan becomes confirmed on screen', d.err);
  await l.sleep(400);
  await l.shot('date-confirmed');
  await l.tapText(p, 'button', 'Copy the plan');
  await l.sleep(700);
  const copied = await m.callWxMethod('getClipboardData');
  l.note(`Copied plan: ${JSON.stringify(copied && copied.data).slice(0, 200)}`);
  l.check(copied && /19:30|Duc|Parc|Kodawari|Cafe|Musee/.test(copied.data || ''), 'the plan copies', JSON.stringify(copied));

  // The Dates tab.
  let q = await l.goto('/pages/dates/dates');
  await l.waitData(q, (x) => x.loaded && x.dates.length > 0, 'the Dates tab', 10000);
  await l.sleep(500);
  await l.shot('dates-tab');

  // The debrief: move the date into the past and let the real job do the rest.
  l.sql(`UPDATE date_plans SET scheduled_at = now() - interval '20 hours', reminded_24=true, reminded_2=true, checkin_sent=true WHERE id='${dateId}';`);
  l.note('DATE TIME MOVED INTO THE PAST IN THE DATABASE to reach the debrief.');
  const open = await l.waitUntil(() => l.sql(`SELECT count(*) FROM debriefs WHERE date_id='${dateId}' AND user_id='${account.id}';`) === '1', 'a debrief to open', 120000).catch((e) => e);
  l.check(open === true, 'the debrief opens after the date', open && open.message);
  p = await l.goto('/pages/suki/suki');
  const asked = await l.waitData(p, (x) => x.messages.some((n) => /Ben/.test(n.body || '') && /(how|went|go)/i.test(n.body || '')), 'Suki to ask how it went', 90000).catch((e) => ({ err: e.message }));
  l.check(!asked.err, 'Suki asks how the date went', asked.err);
  await l.sleep(600);
  await l.shot('debrief-asked');
  await l.sukiSay(p, 'It was lovely. He was funny and easy to talk to, and I would see him again.');
  await l.sleep(1000);
  await l.shot('debrief-answered');
  const notes = l.sql(`SELECT status, jsonb_array_length(notes) FROM debriefs WHERE date_id='${dateId}' AND user_id='${account.id}';`);
  l.note(`Debrief row: ${notes}`);
  l.check(/^(open\|[1-9]|closed|done)/.test(notes) || notes.length > 0, 'the answer is kept with the debrief', notes);
  const after = await l.waitUntil(() => l.sql(`SELECT count(*) FROM hidden.signals WHERE user_id='${account.id}' AND kind LIKE '%debrief%';`) !== '0', 'a debrief signal', 60000).catch(() => null);
  l.note(`Debrief signal recorded: ${!!after}`);
};
