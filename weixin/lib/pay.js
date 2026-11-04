// Buying credits with WeChat Pay. WeChat doesn't allow selling virtual goods in a mini
// program on iOS, so there it offers nothing, and doesn't point anywhere else to pay.
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

function packs() {
  if (platform() === 'ios') return [];
  return PACKS.map((p) => ({ id: p.id, label: t('{n} credits for {price}', { n: p.credits, price: `¥${p.fen / 100}` }) }));
}

const login = () => new Promise((resolve, reject) => {
  wx.login({ success: (r) => resolve(r.code), fail: () => reject(new Error(t("WeChat sign-in didn't work. Please try again."))) });
});

// False if they backed out. WeChat's cancel isn't an error.
const requestPayment = (p) => new Promise((resolve, reject) => {
  wx.requestPayment(Object.assign({}, p, {
    success: () => resolve(true),
    fail: (e) => {
      if (/cancel/.test((e && e.errMsg) || '')) resolve(false);
      else reject(new Error(t("The payment didn't go through. If you weren't charged, you can try again.")));
    },
  }));
});

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

// The credits arrive when WeChat tells the server, which is usually a moment after paying.
async function buy(id) {
  const before = (await api.credits.status()).balance;
  const params = await api.credits.wechatPrepay(id, await login());
  if (!(await requestPayment(params))) return null;
  wx.showToast({ title: t('Paid. Your credits will arrive in a moment.'), icon: 'none' });
  let status = null;
  for (let i = 0; i < 5; i++) {
    await sleep(1500);
    status = await api.credits.status();
    if (status.balance > before) break;
  }
  return status;
}

module.exports = { packs, buy };
