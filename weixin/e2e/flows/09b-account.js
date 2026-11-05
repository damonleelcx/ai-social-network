// Account: password, sessions, export, and deletion on a throwaway. Confirm dialogs and the
// send-file sheet are native, so they're mocked.
const fs = require('fs');
const path = require('path');

module.exports = async (l) => {
  const m = l.m();
  const account = JSON.parse(fs.readFileSync(path.join(l.OUT, 'main-account.json'), 'utf8'));
  await l.signIn(account);
  let p;
  let d;

  // Account: password, sessions, export.
  const other = await l.api('', 'POST', '/v1/auth/signin', { email: account.email, password: account.password });
  const otherToken = other.data.token;
  p = await l.goto('/pages/settings/account/account');
  d = await l.waitData(p, (x) => x.user && x.sessions.length > 1, 'account with two sessions', 10000);
  await l.shot('account-sessions');
  await m.mockWxMethod('showModal', { confirm: true, cancel: false, errMsg: 'showModal:ok' });
  while (d.sessions.some((x) => !x.current)) {
    const chips = await p.$$('.chip.revoke');
    const count = d.sessions.length;
    await chips[d.sessions.findIndex((x) => !x.current)].tap();
    d = await l.waitData(p, (x) => x.sessions.length < count, 'the session to go', 8000);
  }
  await m.restoreWxMethod('showModal');
  d = await p.data();
  l.check((await l.api(otherToken, 'GET', '/v1/settings')).status === 401, 'revoking signs that session out');
  l.check(!!(await m.callWxMethod('getStorageSync', 'token')), 'this session stays signed in');

  await l.tapText(p, '.nav-row', 'Change password');
  await l.sleep(300);
  const next = l.password();
  await l.fill(p, 'input[data-f=current]', 'wrong-password-here');
  await l.fill(p, 'input[data-f=next]', next);
  await l.tapText(p, 'button', 'Change password');
  d = await l.waitData(p, (x) => x.passwordError, 'wrong current password', 8000);
  l.check(!!d.passwordError, 'a wrong current password is refused', d.passwordError);
  l.check(!!(await m.callWxMethod('getStorageSync', 'token')), 'and it does not sign me out');
  await l.fill(p, 'input[data-f=current]', account.password);
  await l.tapText(p, 'button', 'Change password');
  d = await l.waitData(p, (x) => x.passwordDone, 'password changed', 10000);
  await l.shot('password-changed');
  account.password = next;
  fs.writeFileSync(path.join(l.OUT, 'main-account.json'), JSON.stringify(account));
  l.check((await l.api('', 'POST', '/v1/auth/signin', { email: account.email, password: next })).status === 200, 'the new password works');

  await l.tapText(p, 'button', 'Export my data');
  d = await l.waitData(p, (x) => x.exportPath, 'export written', 60000);
  await l.shot('export');
  const size = await m.evaluate((f) => new Promise((res) => { wx.getFileSystemManager().readFile({ filePath: f, encoding: 'utf8', success: (r) => res(r.data), fail: (e) => res(`ERR ${e.errMsg}`) }); }), d.exportPath);
  l.check(typeof size === 'string' && size.startsWith('{'), 'the export is a JSON file on the phone', String(size).slice(0, 80));
  l.check(!/"(score|rank|prior|compat)/i.test(String(size)), 'the export leaks no score or rank');
  l.note(`Export is ${String(size).length} characters.`);
  // Whether the simulator has wx.shareFileMessage decides what this can show.
  const exists = await m.evaluate(() => typeof wx.shareFileMessage);
  l.note(`wx.shareFileMessage in this simulator: ${exists}`);
  if (exists === 'function') {
    await m.mockWxMethod('shareFileMessage', function record(o) { getApp().shared = o.fileName; return { errMsg: 'shareFileMessage:ok' }; }); // eslint-disable-line
    await l.tapText(p, 'button', 'Send the file');
    await l.sleep(500);
    l.check((await m.evaluate(() => getApp().shared)) === 'synagent-data.json', 'the send button offers the file (shareFileMessage mocked)');
    await m.restoreWxMethod('shareFileMessage');
  } else {
    await l.tapText(p, 'button', 'Send the file');
    await l.sleep(500);
    l.check(true, 'the send button does not throw when the simulator lacks it');
  }
  await l.shot('send-file-tapped');

  // Deleting a throwaway account, signed in through the UI.
  const throwaway = await l.person('Throwaway');
  await l.signIn(throwaway);
  p = await l.goto('/pages/settings/account/account');
  await l.waitData(p, (x) => x.user, 'throwaway account');
  await l.tapText(p, 'button', 'Delete my account');
  await l.sleep(300);
  await l.shot('delete-open');
  await l.fill(p, 'input[data-f=deletePassword]', 'not-the-password');
  const del = async () => {
    const buttons = await p.$$('button.danger');
    await buttons[buttons.length - 1].tap();
  };
  await del();
  d = await l.waitData(p, (x) => x.deleteError, 'wrong password on delete', 10000);
  l.check(!!d.deleteError, 'deletion needs the right password', d.deleteError);
  await l.fill(p, 'input[data-f=deletePassword]', throwaway.password);
  await del();
  await l.waitPage('pages/login/login', 15000);
  const row = l.sql(`SELECT deleted_at IS NOT NULL FROM users WHERE id='${throwaway.id}';`);
  l.check(row === 't' || row === '', 'the account is gone', row);
  l.check((await l.api('', 'POST', '/v1/auth/signin', { email: throwaway.email, password: throwaway.password })).status === 401, 'it cannot sign in again');
};
