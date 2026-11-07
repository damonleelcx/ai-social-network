// A real wx.login against the local server. DevTools issues the code, and the server swaps
// it with WeChat, so this needs network and the AppID's credentials on the server.
module.exports = async (l) => {
  const m = l.m();
  await l.signOut();
  const p = await l.goto('/pages/login/login');
  const code = await m.callWxMethod('login');
  l.check(!!(code && code.code), 'wx.login gives a code in the simulator', JSON.stringify(code));

  // If this WeChat identity is new, the terms modal needs an answer, so it's mocked to yes.
  await m.mockWxMethod('showModal', { confirm: true, cancel: false, errMsg: 'showModal:ok' });
  const before = Number(l.sql("SELECT count(*) FROM users WHERE email LIKE 'wechat-%'"));
  await l.tapText(p, 'button', (await p.data()).t.wechat);
  const outcome = await l.waitUntil(async () => {
    const c = await l.page();
    if (c.path === 'pages/suki/suki') return { ok: true };
    const d = await c.data();
    return d.error ? { error: d.error } : null;
  }, 'WeChat sign-in to finish', 30000);
  await m.restoreWxMethod('showModal');
  l.check(outcome.ok, 'WeChat sign-in reaches Suki', outcome.error);
  const after = Number(l.sql("SELECT count(*) FROM users WHERE email LIKE 'wechat-%'"));
  l.note(`WeChat accounts before ${before}, after ${after}`);
  await l.sleep(1000);
  await l.shot('wechat-signed-in');
};
