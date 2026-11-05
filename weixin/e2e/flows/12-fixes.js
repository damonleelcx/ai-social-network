// Checks for bugs these tests found: a posted card stays posted, the delete panel comes into
// view, and whole-photo preview really opens in the simulator.
const fs = require('fs');
const path = require('path');

module.exports = async (l) => {
  const m = l.m();
  const account = JSON.parse(fs.readFileSync(path.join(l.OUT, 'main-account.json'), 'utf8'));
  await l.signIn(account);

  let p = await l.goto('/pages/suki/suki');
  await l.waitData(p, (d) => d.loaded && d.messages.length > 0, 'Suki', 10000);
  await l.sleep(2000);
  let postCards = 0;
  let stillOffered = 0;
  for (const c of await p.$$('wx-suki-card')) {
    const d = await c.data();
    if (d.card && d.card.kind === 'buddy_post') {
      postCards += 1;
      if (d.state !== 'done') stillOffered += 1;
      l.check(d.activityText === 'Movie', 'the post card names the plan in words', d.activityText);
    }
  }
  l.check(postCards > 0, 'the conversation has a post card', postCards);
  l.check(stillOffered === 0, 'a post already made is not offered again', stillOffered);

  p = await l.goto('/pages/dates/dates');
  const dates = await l.waitData(p, (x) => x.loaded && x.posts.length > 0, 'my posts', 10000);
  l.check(dates.posts[0].kindText === 'Movie', 'my posts name the plan in words', dates.posts[0].kindText);

  p = await l.goto('/pages/settings/account/account');
  await l.waitData(p, (x) => x.user, 'account');
  const danger = await p.$$('button.danger');
  await danger[0].tap();
  await l.sleep(1000);
  const top = await p.scrollTop();
  l.check(top > 0, 'opening the delete panel scrolls it into view', top);
  await l.shot('delete-panel-in-view');

  // A real wx.previewImage, with nothing mocked. Automation can't read a native overlay, so
  // the screenshot is the evidence.
  p = await l.goto('/pages/me/me');
  await l.waitData(p, (x) => x.photos.length > 0 && x.photos[0].path, 'my photo', 15000);
  await (await p.$('.photo-wrap .photo')).tap();
  await l.sleep(1800);
  await l.shot('real-preview');
  await l.goto('/pages/me/me');
};
