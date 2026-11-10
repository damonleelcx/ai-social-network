// Buying credits. Credits are virtual goods, so WeChat requires its virtual payment
// (wx.requestVirtualPayment) for them on every phone, iPhone included, and nothing else: no
// ordinary WeChat Pay, and never a pointer to pay anywhere else. If the server hasn't turned
// virtual payment on (credits.wechat_virtual), nothing is for sale.
const api = require('./api');
const { t, modalButtons } = require('./i18n');

const PACKS = [
  { id: 'small', credits: 100, fen: 3500 },
  { id: 'medium', credits: 300, fen: 8800 },
  { id: 'large', credits: 1000, fen: 23800 },
];

// Call after api.credits.status(), which says whether virtual payment is on.
const label = (p) => t('{n} credits for {price}', { n: p.credits, price: `¥${p.fen / 100}` });

function packs() {
  if (!api.credits.virtual) return [];
  return PACKS.map((p) => ({ id: p.id, label: label(p) }));
}

// WeChat's rules want a clear notice on the page before any payment (运营规范 2.14), so a tap
// on a price only asks; nothing is ordered until they say yes.
const NOTICE = 'Credits are virtual goods used for messages to Suki. You pay with WeChat virtual payment.';

const confirmed = (id) => new Promise((resolve) => {
  const pack = PACKS.find((p) => p.id === id);
  wx.showModal(Object.assign({
    title: t('Confirm your purchase'),
    content: `${pack ? label(pack) : ''}\n${t(NOTICE)}`,
    success: (r) => resolve(!!r.confirm),
    fail: () => resolve(false),
  }, modalButtons()));
});

const login = () => new Promise((resolve, reject) => {
  wx.login({ success: (r) => resolve(r.code), fail: () => reject(new Error(t("WeChat sign-in didn't work. Please try again."))) });
});

const failed = () => new Error(t("The payment didn't go through. If you weren't charged, you can try again."));
const cancelled = (e) => /cancel/i.test((e && e.errMsg) || '') || (e && e.errCode === -2);

// False if they backed out. WeChat's cancel isn't an error.
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
// If that's slow, the server asks WeChat about the order itself.
async function buy(id) {
  const before = (await api.credits.status()).balance;
  if (!api.credits.virtual) throw new Error(t("Credits can't be bought here right now. You can still use any you already have."));
  if (!(await confirmed(id))) return null;
  const o = await api.credits.wechatVirtual(id, await login());
  const order = o.order;
  if (!(await requestVirtualPayment(o))) return null;
  wx.showToast({ title: t('Paid. Your credits will arrive in a moment.'), icon: 'none' });
  let status = null;
  for (let i = 0; i < 6; i++) {
    await sleep(1500);
    if (i >= 2) await api.credits.wechatVirtualCheck(order).catch(() => {});
    status = await api.credits.status();
    if (status.balance > before) break;
  }
  return status;
}

module.exports = { packs, buy, NOTICE };
