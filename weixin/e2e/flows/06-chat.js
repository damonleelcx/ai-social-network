// Chat with a match, coaching hints with Ask Suki (real model), report and block.
// Action sheets and confirm dialogs are native, so they're mocked to answer.
const fs = require('fs');
const path = require('path');

module.exports = async (l) => {
  const m = l.m();
  const account = JSON.parse(fs.readFileSync(path.join(l.OUT, 'main-account.json'), 'utf8'));
  const world = JSON.parse(fs.readFileSync(path.join(l.OUT, 'world.json'), 'utf8'));
  await l.signIn(account);
  const ben = world.ben;

  let p = await l.goto('/pages/matches/matches');
  await l.waitData(p, (d) => d.loaded && d.matches.length > 0, 'a match to open', 15000);
  await l.tapText(p, '.card.row', 'Ben');
  p = await l.waitPage('pages/chat/chat');
  let d = await l.waitData(p, (x) => x.match && x.match.status === 'active', 'the chat', 15000);
  await l.sleep(800);
  await l.shot('chat-empty');
  l.check(d.placeholder.includes('Ben'), 'the composer names them', d.placeholder);

  // Ana writes, and it's in the conversation on the server.
  const hello = `Hi Ben, which hike do you like best? (${Date.now()})`;
  await l.fill(p, '.composer input', hello);
  await (await p.$('.composer .send')).tap();
  d = await l.waitData(p, (x) => x.messages.some((n) => n.body === hello), 'my message to show', 10000);
  l.check(d.draft === '', 'sending clears the box');
  const seen = await l.api(ben.token, 'GET', `/v1/matches/${world.match}/messages`);
  l.check((seen.data.messages || []).some((n) => n.body.includes('which hike')), 'the other person receives it', JSON.stringify(seen.data).slice(0, 200));

  // Ben answers. The socket or the 10 second poll brings it in.
  await l.api(ben.token, 'POST', `/v1/matches/${world.match}/messages`, { text: 'Fontainebleau, easily. Do you climb?', client_id: `e2e-${Date.now()}` });
  d = await l.waitData(p, (x) => x.messages.some((n) => (n.body || '').includes('Fontainebleau')), 'their reply to arrive', 15000).catch((e) => ({ err: e.message }));
  l.check(!d.err, 'their reply shows without reloading', d.err);
  await l.sleep(400);
  await l.shot('chat-messages');

  // Ask Suki, with the real model. A match already starts with a hint from her, so this
  // waits for a new one.
  await l.sleep(500);
  const hintBefore = (await p.data()).hint;
  await (await p.$('.ask-row .chip')).tap();
  d = await l.waitData(p, (x) => x.asking, 'the panel to say she is thinking', 4000);
  await l.sleep(500);
  await l.shot('chat-asking');
  d = await l.waitData(p, (x) => x.hint && (!hintBefore || x.hint.id !== hintBefore.id), 'a new coaching hint', 120000).catch((e) => ({ err: e.message }));
  l.check(!d.err, 'Ask Suki brings back a private hint', d.err);
  if (!d.err) {
    l.note(`Hint: ${JSON.stringify(d.hint)}`);
    await l.sleep(400);
    await l.shot('chat-hint');
    l.check(d.asking === false, 'the thinking line clears');
    const theirs = await l.api(ben.token, 'GET', `/v1/matches/${world.match}/hints`);
    l.check(!(theirs.data.hints || []).some((h) => h.id === d.hint.id), 'the other person does not get this hint');
    const leak = (await l.api(ben.token, 'GET', `/v1/matches/${world.match}/messages`)).data.messages || [];
    l.check(!leak.some((n) => n.body === d.hint.body), 'the hint is not in the shared conversation');
    if ((d.hint.suggestions || []).length) {
      const before = (await l.api(ben.token, 'GET', `/v1/matches/${world.match}/messages`)).data.messages.length;
      await (await p.$('.suggestion')).tap();
      d = await l.waitData(p, (x) => x.draft, 'the suggestion in the box', 3000);
      l.check(d.draft === d.hint.suggestions[0], 'a suggestion fills the box');
      await l.sleep(500);
      await l.shot('chat-suggestion-typing');
      const after = (await l.api(ben.token, 'GET', `/v1/matches/${world.match}/messages`)).data.messages.length;
      l.check(after === before, 'a suggestion sends nothing by itself');
      await (await p.$('.composer input')).trigger('blur', { value: d.draft });
      await l.sleep(300);
      await l.fill(p, '.composer input', '');
    }
    const shown = d.hint.id;
    const close = await p.$('.coach-close');
    await close.tap();
    await l.sleep(1200);
    d = await p.data();
    l.check(!d.hint || d.hint.id !== shown, 'dismissing clears it or moves on to the next');
    const stored = l.sql(`SELECT dismissed_at IS NOT NULL FROM coach_hints WHERE id='${shown}';`);
    l.check(stored === 't', 'the dismissal is saved', stored);
  }

  // The person page from the chat bar.
  await (await p.$('.bar .grow')).tap();
  let q = await l.waitPage('pages/person/person');
  await l.waitData(q, (x) => x.person, 'their profile');
  await l.sleep(700);
  await l.shot('person-from-chat');
  await m.navigateBack();
  await l.sleep(600);

  // Report and block on a second match, so the first stays for date planning.
  const token = await m.callWxMethod('getStorageSync', 'token');
  const proposals = (await l.api(token, 'GET', '/v1/proposals')).data.proposals || [];
  const cal = proposals.find((x) => x.person.display_name === 'Cal');
  if (!cal || cal.status !== 'open') {
    l.note('Cal was already decided on an earlier run, so report and block did not run again.');
    return;
  }
  await l.api(token, 'POST', `/v1/proposals/${cal.id}/decide`, { decision: 'accept' });
  const made = await l.api(world.cal.token, 'POST', `/v1/proposals/${cal.id}/decide`, { decision: 'accept' });
  const calMatch = made.data.match_id;
  l.check(!!calMatch, 'a second match to report and block', JSON.stringify(made.data));
  p = await l.goto(`/pages/chat/chat?id=${calMatch}`);
  await l.waitData(p, (x) => x.match && x.match.status === 'active', 'second chat', 10000);

  // More, then Report, then a reason. The same mocked answer picks both.
  await m.mockWxMethod('showActionSheet', { tapIndex: 1, errMsg: 'showActionSheet:ok' });
  await l.tapText(p, '.bar .chip', 'More');
  await l.sleep(1500);
  await m.restoreWxMethod('showActionSheet');
  const reports = l.sql(`SELECT reason FROM reports WHERE subject_id='${world.cal.id}' ORDER BY created_at DESC LIMIT 1;`);
  l.check(reports === 'fake_profile', 'a report reaches the server with its reason', reports);

  await m.mockWxMethod('showActionSheet', { tapIndex: 2, errMsg: 'showActionSheet:ok' });
  await m.mockWxMethod('showModal', { confirm: true, cancel: false, errMsg: 'showModal:ok' });
  await l.tapText(p, '.bar .chip', 'More');
  await l.sleep(2000);
  await m.restoreWxMethod('showActionSheet');
  await m.restoreWxMethod('showModal');
  const blocked = (await l.api(token, 'GET', '/v1/blocks')).data.blocked || [];
  l.check(blocked.some((b) => b.user_id === world.cal.id), 'blocking from the chat is saved', JSON.stringify(blocked));
  const status = l.sql(`SELECT status FROM matches WHERE id='${calMatch}';`);
  l.check(status !== 'active', 'blocking ends the match', status);
  const cantSend = await l.api(world.cal.token, 'POST', `/v1/matches/${calMatch}/messages`, { text: 'hello?', client_id: `e2e-${Date.now()}` });
  l.check(cantSend.status >= 400, 'the blocked person cannot write any more', cantSend.status);
};
