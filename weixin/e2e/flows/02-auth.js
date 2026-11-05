// Email sign-up through forgot password, in the UI, in English.
const fs = require('fs');
const path = require('path');

module.exports = async (l) => {
  const m = l.m();
  const email = `e2e-auth-${Date.now()}@example.com`;
  const first = l.password();
  const second = l.password();

  await l.signOut();
  let p = await l.goto('/pages/login/login');
  await m.mockWxMethod('showActionSheet', { tapIndex: 1, errMsg: 'showActionSheet:ok' });
  await (await l.waitUntil(() => p.$('.hero .link'), 'language link')).tap();
  await l.sleep(300);
  await m.restoreWxMethod('showActionSheet');
  await l.tapText(p, '.link', 'Sign in with email');
  await l.tapText(p, '.link', 'No account yet');
  await l.waitData(p, (d) => d.mode === 'signup', 'signup mode');
  await l.shot('signup');

  await l.fill(p, 'input[data-f=name]', 'Auth Test');
  await l.fill(p, 'input[data-f=email]', email);
  await l.fill(p, 'input[data-f=password]', first);
  // Tapping submit without the box ticked must be refused.
  await l.tapText(p, 'button', 'Create my account');
  let d = await l.waitData(p, (x) => x.error, 'terms error');
  l.check(/accept the terms/i.test(d.error), 'sign-up refuses without the terms box', d.error);

  await (await p.$('checkbox-group')).trigger('change', { value: ['yes'] });
  await l.tapText(p, 'button', 'Create my account');
  d = await l.waitData(p, (x) => x.mode === 'verify', 'verify step');
  l.check(true, 'sign-up reaches the code step');
  await l.shot('verify');

  const userId = l.sql(`SELECT id FROM users WHERE email='${email}'`);
  l.check(!!userId, 'the account exists in the database');

  await l.fill(p, 'input[data-f=code]', '000000');
  l.setCode(userId, 'verify', '111111');
  await l.tapText(p, 'button', 'Verify');
  d = await l.waitData(p, (x) => x.error, 'wrong code error');
  l.check(/wrong|expired/i.test(d.error), 'a wrong code is refused with a reason', d.error);
  await l.shot('wrong-code');

  await l.fill(p, 'input[data-f=code]', '111111');
  await l.tapText(p, 'button', 'Verify');
  p = await l.waitPage('pages/suki/suki');
  l.check(true, 'the right code goes to Suki');
  l.check(l.sql(`SELECT email_verified_at IS NOT NULL FROM users WHERE id='${userId}'`) === 't', 'email marked verified');
  await l.sleep(1500);
  await l.shot('suki-first');

  // Sign out.
  p = await l.goto('/pages/settings/account/account');
  await l.waitData(p, (x) => x.user, 'account loaded');
  await l.shot('account');
  await l.tapText(p, 'button', 'Sign out');
  p = await l.waitPage('pages/login/login');
  l.check(true, 'sign-out returns to login');
  const stored = await m.callWxMethod('getStorageSync', 'token');
  l.check(!stored, 'sign-out clears the token', stored);

  // Wrong password, then the right one.
  await l.tapText(p, '.link', 'Sign in with email');
  await l.fill(p, 'input[data-f=email]', email);
  await l.fill(p, 'input[data-f=password]', 'definitely-not-it');
  await l.tapText(p, 'button', 'Sign in');
  d = await l.waitData(p, (x) => x.error, 'wrong password error');
  l.check(/invalid|password/i.test(d.error), 'a wrong password is refused with a reason', d.error);
  await l.shot('wrong-password');
  await l.fill(p, 'input[data-f=password]', first);
  await l.tapText(p, 'button', 'Sign in');
  p = await l.waitPage('pages/suki/suki');
  l.check(true, 'the right password signs in');

  // Forgot password, through to the new one working.
  p = await l.goto('/pages/settings/account/account');
  await l.waitData(p, (x) => x.user, 'account loaded');
  await l.tapText(p, 'button', 'Sign out');
  p = await l.waitPage('pages/login/login');
  await l.tapText(p, '.link', 'Sign in with email');
  await l.tapText(p, '.link', 'I forgot my password');
  await l.waitData(p, (x) => x.mode === 'forgot', 'forgot mode');
  await l.shot('forgot');
  await l.fill(p, 'input[data-f=email]', email);
  await l.tapText(p, 'button', 'Send me a code');
  d = await l.waitData(p, (x) => x.mode === 'reset', 'reset step');
  l.check(d.resetLead.includes(email), 'the code step names the email', d.resetLead);
  await l.shot('reset');
  l.setCode(userId, 'reset', '222222');
  await l.fill(p, 'input[data-f=code]', '999999');
  await l.fill(p, 'input[data-f=password]', second);
  await l.tapText(p, 'button', 'Set new password');
  d = await l.waitData(p, (x) => x.error, 'wrong reset code');
  l.check(/wrong|expired/i.test(d.error), 'a wrong reset code is refused', d.error);
  await l.fill(p, 'input[data-f=code]', '222222');
  await l.tapText(p, 'button', 'Set new password');
  d = await l.waitData(p, (x) => x.mode === 'signin', 'back to sign in');
  l.check(d.email === email, 'back at sign-in with the email kept');
  await l.fill(p, 'input[data-f=password]', first);
  await l.tapText(p, 'button', 'Sign in');
  d = await l.waitData(p, (x) => x.error, 'old password refused');
  l.check(true, 'the old password no longer works');
  await l.fill(p, 'input[data-f=password]', second);
  await l.tapText(p, 'button', 'Sign in');
  await l.waitPage('pages/suki/suki');
  l.check(true, 'the new password signs in');

  fs.writeFileSync(path.join(l.OUT, 'main-account.json'), JSON.stringify({ email, password: second, id: userId }));
};
