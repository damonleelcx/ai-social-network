// Buying credits. Credits are virtual goods, so WeChat requires its virtual payment
// (wx.requestVirtualPayment) for them, and it has closed ordinary WeChat Pay for them on
// Android. The server says which is on (credits.wechat_virtual); ordinary WeChat Pay is kept
// for a server without virtual payment, and offers nothing on iOS. It never points anywhere
// else to pay: WeChat forbids that.
const api = require('./api');
const { t } = require('./i18n');

const PACKS = [
  { id: 'small', credits: 100, fen: 3500 },
  { id: 'medium', credits: 300, fen: 8800 },
  { id: 'large', credits: 1000, fen: 23800 },
];

function platform() {
  const info = wx.getDeviceInfo ? wx.getDeviceInfo() : wx.getSystemInfoSync();
  return info.platform;
}

// Call after api.credits.status(), which says whether virtual payment is on.
function packs() {
  if (!api.credits.virtual && platform() === 'ios') return [];
  return PACKS.map((p) => ({ id: p.id, label: t('{n} credits for {price}', { n: p.credits, price: `¥${p.fen / 100}` }) }));
}

const login = () => new Promise((resolve, reject) => {
  wx.login({ success: (r) => resolve(r.code), fail: () => reject(new Error(t("WeChat sign-in didn't work. Please try again."))) });
});

const failed = () => new Error(t("The payment didn't go through. If you weren't charged, you can try again."));
const cancelled = (e) => /cancel/i.test((e && e.errMsg) || '') || (e && e.errCode === -2);

// False if they backed out. WeChat's cancel isn't an error.
const requestPayment = (p) => new Promise((resolve, reject) => {
  wx.requestPayment(Object.assign({}, p, {
    success: () => resolve(true),
    fail: (e) => (cancelled(e) ? resolve(false) : reject(failed())),
  }));
});

const requestVirtualPayment = (o) => new Promise((resolve, reject) => {
  if (!wx.requestVirtualPayment) {
    reject(new Error(t('Please update WeChat to buy credits.')));
    return;
  }
  // signData goes through exactly as the server signed it.
  wx.requestVirtualPayment({
    mode: o.mode,
    signData: o.signData,
    paySig: o.paySig,
    signature: o.signature,
    success: () => resolve(true),
    fail: (e) => (cancelled(e) ? resolve(false) : reject(failed())),
  });
});

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// The credits arrive when WeChat tells the server, which is usually a moment after paying.
// With virtual payment, if that's slow, the server asks WeChat about the order itself.
async function buy(id) {
  const before = (await api.credits.status()).balance;
  let order = null;
  if (api.credits.virtual) {
    const o = await api.credits.wechatVirtual(id, await login());
    order = o.order;
    if (!(await requestVirtualPayment(o))) return null;
  } else {
    const params = await api.credits.wechatPrepay(id, await login());
    if (!(await requestPayment(params))) return null;
  }
  wx.showToast({ title: t('Paid. Your credits will arrive in a moment.'), icon: 'none' });
  let status = null;
  for (let i = 0; i < 6; i++) {
    await sleep(1500);
    if (order && i >= 2) await api.credits.wechatVirtualCheck(order).catch(() => {});
    status = await api.credits.status();
    if (status.balance > before) break;
  }
  return status;
}

module.exports = { packs, buy };
