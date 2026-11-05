// Buddy posts: Ana posts through Suki's card (real model), Ben offers, Ana picks. Then Ben
// posts, and Ana sees it through Suki and joins.
const fs = require('fs');
const path = require('path');

const cardWith = async (p, words) => {
  const cards = await p.$$('wx-suki-card');
  for (let i = cards.length - 1; i >= 0; i--) {
    if ((await cards[i].wxml()).includes(words)) return cards[i];
  }
  return null;
};

module.exports = async (l) => {
  const m = l.m();
  const account = JSON.parse(fs.readFileSync(path.join(l.OUT, 'main-account.json'), 'utf8'));
  const world = JSON.parse(fs.readFileSync(path.join(l.OUT, 'world.json'), 'utf8'));
  await l.signIn(account);
  const token = await m.callWxMethod('getStorageSync', 'token');
  const dan = world.dan; // Ben and Ana already match, and an existing match can rule people out.

  let p = await l.goto('/pages/suki/suki');
  await l.waitData(p, (d) => d.loaded, 'Suki loaded');
  await l.sukiSay(p, 'I want someone to see a movie with on Saturday evening. Open to anyone. Please post it for me on the buddy board for 48 hours.');
  let d = await p.data();
  const posted = d.messages.some((x) => (x.cards || []).some((c) => c.kind === 'buddy_post'));
  l.note(`Suki said: ${d.messages[d.messages.length - 1].body}`);
  l.check(posted, 'Suki offers a post card');
  if (posted) {
    await l.sleep(800);
    await l.shot('buddy-post-card');
    const card = await l.waitUntil(() => cardWith(p, 'Post it'), 'the post card');
    await (await card.$('button')).tap();
    await l.sleep(1500);
    await l.shot('buddy-posted');
  }
  const mine = (await l.api(token, 'GET', '/v1/buddies/mine')).data.requests || [];
  l.check(mine.length > 0, 'the post is on the server', JSON.stringify(mine).slice(0, 200));
  if (mine.length) {
    const offer = await l.api(dan.token, 'POST', `/v1/buddies/${mine[0].id}/offer`);
    l.check(offer.status === 200, 'another person joins it', `${offer.status} ${JSON.stringify(offer.data)}`);
    p = await l.goto('/pages/dates/dates');
    d = await l.waitData(p, (x) => x.loaded && x.posts.length > 0 && x.posts[0].offers.length > 0, 'the offer on the Dates tab', 10000).catch((e) => ({ err: e.message }));
    l.check(!d.err, 'the offer shows on the Dates tab', d.err);
    await l.sleep(500);
    await l.shot('buddy-offer');
    if (!d.err) {
      await m.mockWxMethod('showModal', { confirm: true, cancel: false, errMsg: 'showModal:ok' });
      await (await p.$('.row.gap .chip')).tap();
      await l.sleep(1800);
      await m.restoreWxMethod('showModal');
      const c = await l.page();
      l.check(c.path === 'pages/chat/chat', 'picking someone opens a chat with them', c.path);
      await l.sleep(800);
      await l.shot('buddy-chat');
      const data = await c.data();
      l.check(/Buddy|Movie|movie/.test(data.barText), 'the chat says it is a buddy plan', data.barText);
      clearTimeout(0);
    }
  }

  // Dan posts. Ana asks Suki about it.
  const post = await l.api(dan.token, 'POST', '/v1/buddies', { kind: 'meal', when: 'Friday night', note: 'Ramen, anyone?', expires_in_hours: 48, same_gender: false });
  l.check(post.status < 300, 'someone else posts a plan', `${post.status} ${JSON.stringify(post.data).slice(0, 200)}`);
  p = await l.goto('/pages/suki/suki');
  await l.waitData(p, (x) => x.loaded, 'Suki loaded');
  await l.sukiSay(p, 'Is anyone around looking for company for a meal? Show me the buddy board.');
  await l.sleep(800);
  await l.shot('buddy-board');
  d = await p.data();
  const reqCard = d.messages.some((x) => (x.cards || []).some((c) => c.kind === 'buddy_request'));
  l.note(`Suki said: ${d.messages[d.messages.length - 1].body}`);
  l.check(reqCard, 'Suki shows the post as a card');
  if (reqCard) {
    const card = await l.waitUntil(() => cardWith(p, 'Ramen'), 'the request card', 8000).catch(() => null);
    if (card) {
      await (await card.$('button')).tap();
      await l.sleep(1200);
      await l.shot('buddy-joined');
      const r = await l.api(token, 'GET', `/v1/buddies/${post.data.request.id}`);
      l.check(r.data.request && r.data.request.offered === true, 'joining through the card is saved', JSON.stringify(r.data).slice(0, 200));
    } else {
      l.check(false, 'the request card shows its note');
    }
  }
};
