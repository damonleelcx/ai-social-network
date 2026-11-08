// The daily check-in on Me, and what sharing sends. A new person checks in, taps again
// (nothing more is paid), and the share messages carry their invite code and a fixed
// image, with no word about credits or any reward (WeChat bans 诱导分享).
const REWARD = /credit|积分|크레딧|クレジット|crédito|奖|reward|보상|報酬|recompensa|免费|free|gratis/i;

module.exports = async (l) => {
  const lin = await l.person('Lin');
  await l.signIn(lin);
  let p = await l.goto('/pages/me/me');
  let d = await l.waitData(p, (x) => x.checkin && x.invite, 'the check-in and invite cards', 15000);
  l.check(!d.checkin.checked_in && d.checkin.credits > 0, 'a new person has not checked in today', JSON.stringify(d.checkin));
  const before = d.credits.balance;
  const code = d.invite.code;
  await l.shot('me-checkin-before');

  const button = await p.$('.checkin');
  await button.tap();
  d = await l.waitData(p, (x) => x.checkin && x.checkin.checked_in && x.credits.balance > before, 'the check-in to land', 15000);
  l.check(d.credits.balance === before + d.checkin.credits, 'checking in adds the day’s credits', `${before} -> ${d.credits.balance}`);
  l.check(d.usage.length > 0 && d.usage[0].delta === d.checkin.credits, 'the check-in shows in the credits history', JSON.stringify(d.usage[0]));
  await l.shot('me-checkin-done');

  // A second tap, even straight at the server, pays nothing more today.
  const again = await l.api(lin.token, 'POST', '/v1/checkin', {});
  l.check(again.status === 200 && again.data.earned === 0, 'checking in twice pays once', JSON.stringify(again.data));

  const msg = await p.callMethod('onShareAppMessage');
  l.check(msg.path === `/pages/suki/suki?ref=${code}` && /assets\/share\//.test(msg.imageUrl), 'sharing from Me sends Suki with the invite code and a fixed image', JSON.stringify(msg));
  l.check(!REWARD.test(msg.title), 'the share title mentions no reward', msg.title);

  p = await l.goto('/pages/suki/suki');
  await l.sleep(1000);
  const tl = await p.callMethod('onShareTimeline');
  l.check(tl.query === `ref=${code}` && /assets\/share\//.test(tl.imageUrl), 'sharing to Moments carries the invite code and a fixed image', JSON.stringify(tl));
  l.check(!REWARD.test(tl.title), 'the Moments title mentions no reward', tl.title);
};
