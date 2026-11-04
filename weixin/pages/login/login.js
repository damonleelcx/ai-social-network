const api = require('../../lib/api');
const live = require('../../lib/live');
const { confirm } = require('../../lib/format');

const share = require('../../lib/share');
const NEEDS_TERMS = '请先同意用户协议';
const wxLogin = () => new Promise((resolve, reject) => {
  wx.login({ success: (r) => resolve(r.code), fail: () => reject(new Error('微信登录失败，请再试一次。')) });
});

Page({
  data: { mode: 'wechat', name: '', email: '', password: '', code: '', accepted: false, busy: false, error: '' },
  onLoad(q) {
    if (q.mode) this.setData({ mode: q.mode });
    else if (api.getToken()) this.enter();
  },
  onInput(e) {
    this.setData({ [e.currentTarget.dataset.f]: e.detail.value });
  },
  onTerms(e) {
    this.setData({ accepted: e.detail.value.length > 0 });
  },
  onLegal(e) {
    // A web-view needs a filed business domain, so the address goes to the clipboard.
    wx.setClipboardData({ data: `${api.API_URL}/legal/${e.currentTarget.dataset.page}?lang=zh` });
  },
  onSwitch() {
    this.setData({ mode: this.data.mode === 'signup' ? 'signin' : 'signup', error: '' });
  },
  onMode(e) {
    this.setData({ mode: e.currentTarget.dataset.mode, error: '' });
  },
  async act(fn) {
    if (this.data.busy) return;
    this.setData({ busy: true, error: '' });
    try {
      await fn();
    } catch (e) {
      this.setData({ error: e.message });
    }
    this.setData({ busy: false });
  },
  enter() {
    live.connect();
    wx.switchTab({ url: '/pages/suki/suki' });
  },
  after(res) {
    api.setToken(res.token);
    share.clearPending();
    if (res.user.email_verified) this.enter();
    else this.setData({ mode: 'verify', password: '' });
  },
  onSubmit() {
    const { mode, name, email, password, accepted } = this.data;
    this.act(async () => {
      if (mode === 'signin') return this.after(await api.auth.signIn(email.trim(), password));
      if (!accepted) throw new Error('请先勾选同意用户协议');
      this.after(await api.auth.signUp({ email: email.trim(), password, name: name.trim(), accept_terms: true, ref: share.pending(), language: 'zh' }));
    });
  },
  onVerify() {
    this.act(async () => {
      await api.auth.verify(this.data.code.trim());
      this.enter();
    });
  },
  onWeChat() {
    this.act(async () => {
      // A code works once, so each try gets its own.
      try {
        this.after(await api.auth.wechat(await wxLogin(), false, share.pending()));
      } catch (e) {
        if (e.message !== NEEDS_TERMS) throw e;
        if (!(await confirm('欢迎来到 SynAgent', '继续即表示你已年满 18 岁，并同意用户协议和隐私政策。'))) return;
        this.after(await api.auth.wechat(await wxLogin(), true, share.pending()));
      }
    });
  },
  onResend() {
    this.act(async () => {
      await api.auth.resend();
      wx.showToast({ title: '已发送', icon: 'none' });
    });
  },
  onLeave() {
    api.setToken('');
    this.setData({ mode: 'signin', code: '' });
  },
  onForgot() {
    this.act(async () => {
      await api.auth.forgot(this.data.email.trim());
      this.setData({ mode: 'reset', password: '', code: '' });
    });
  },
  onReset() {
    const { email, code, password } = this.data;
    this.act(async () => {
      await api.auth.reset(email.trim(), code.trim(), password);
      this.setData({ mode: 'signin', password: '', code: '' });
      wx.showToast({ title: '密码已重设', icon: 'none' });
    });
  },
});
