// The picker before sign-in, and that the words change with it. The action sheet is native,
// so it's mocked to answer with the language's position.
const ORDER = { zh: 0, en: 1, ko: 2, ja: 3, es: 4 };
const SIGN_IN = { zh: '微信', en: 'WeChat', ko: 'WeChat', ja: 'WeChat', es: 'WeChat' };
const EMAIL = { zh: '邮箱', en: 'email', ko: '이메일', ja: 'メール', es: 'correo' };

module.exports = async (l) => {
  const m = l.m();
  // Leave whoever is signed in. wx.reLaunch to login with no token shows the picker.
  await l.signOut();
  const p = await l.goto('/pages/login/login');
  for (const lang of ['ko', 'ja', 'es', 'en', 'zh']) {
    await m.mockWxMethod('showActionSheet', { tapIndex: ORDER[lang], errMsg: 'showActionSheet:ok' });
    await (await p.$('.hero .link')).tap();
    await l.sleep(400);
    const d = await l.waitData(p, (x) => x.t.wechat.includes(SIGN_IN[lang]) && x.t.useEmail.includes(EMAIL[lang]), `login in ${lang}`, 4000).catch((e) => ({ err: e.message }));
    l.check(!d.err, `login page words change to ${lang}`, d.err);
    await l.shot(`login-${lang}`);
    const raw = JSON.stringify((await p.data()).firstTime);
    l.check(!/\{terms\}|\{privacy\}/.test(raw), `terms line has no raw placeholder in ${lang}`, raw);
  }
  await m.restoreWxMethod('showActionSheet');
};
