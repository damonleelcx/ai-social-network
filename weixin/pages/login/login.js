const api = require('../../lib/api');
const live = require('../../lib/live');
const i18n = require('../../lib/i18n');
const settings = require('../../lib/settings');
const { confirm } = require('../../lib/format');

const share = require('../../lib/share');

const { t } = i18n;

const COPY = {
  tagline: 'Suki introduces you to people, and finds you company for plans',
  language: 'Language',
  welcomeBack: "There you are. I was hoping you'd come back.",
  wechat: 'Continue with WeChat',
  useEmail: 'Sign in with email',
  name: 'Name',
  email: 'Email',
  password: 'Password',
  newAccountPassword: 'Password (at least 10 characters)',
  signUp: 'Create my account',
  signIn: 'Sign in',
  haveAccount: 'I already have an account',
  noAccount: 'No account yet? Sign up',
  forgot: 'I forgot my password',
  sentCode: 'We sent a code to your email.',
  code: 'Six digit code',
  verify: 'Verify',
  resend: 'Send a new code',
  otherAccount: 'Use a different account',
  forgotLead: "It happens. Tell me your email and I'll send you a code.",
  sendCode: 'Send me a code',
  back: 'Back to sign in',
  newPassword: 'New password',
  atLeast: 'At least 10 characters.',
  setPassword: 'Set new password',
  otherEmail: 'Use a different email',
};
const FIRST_TIME = "The first time you sign in, you confirm you're 18 or older and agree to the {terms} and {privacy}.";
const SIGN_UP_TERMS = "I'm 18 or older, and I agree to the {terms} and {privacy}.";

const wxLogin = () => new Promise((resolve, reject) => {
  wx.login({ success: (r) => resolve(r.code), fail: () => reject(new Error(t("WeChat sign-in didn't work. Please try again."))) });
});

// The links sit in a different place in each language's sentence.
function legalParts(sentence) {
  return t(sentence).split(/(\{terms\}|\{privacy\})/).filter(Boolean).map((text) => {
    if (text === '{terms}') return { text: t('terms'), page: 'terms' };
    if (text === '{privacy}') return { text: t('privacy policy'), page: 'privacy' };
    return { text, page: '' };
  });
}

Page({
  data: { t: i18n.strings(COPY), mode: 'wechat', name: '', email: '', password: '', code: '', accepted: false, busy: false, error: '', firstTime: [], signUpTerms: [], resetLead: '' },
  onLoad(q) {
    if (q.mode) this.setData({ mode: q.mode });
    else if (api.getToken()) this.enter();
  },
  onShow() {
    i18n.apply(this, COPY);
    this.load();
  },
  load() {
    this.setData({ firstTime: legalParts(FIRST_TIME), signUpTerms: legalParts(SIGN_UP_TERMS), resetLead: this.resetLead() });
  },
  resetLead() {
    return t('If {value} has an account, a 6-digit code is on its way. Enter it with your new password.', { value: this.data.email.trim() });
  },
  onLanguage() {
    const list = i18n.LANGUAGES;
    wx.showActionSheet({
      itemList: list.map((l) => l.label),
      success: (r) => i18n.setLanguage(list[r.tapIndex].value),
    });
  },
  onInput(e) {
    this.setData({ [e.currentTarget.dataset.f]: e.detail.value });
  },
  onTerms(e) {
    this.setData({ accepted: e.detail.value.length > 0 });
  },
  onLegal(e) {
    const page = e.currentTarget.dataset.page;
    // A web-view needs a filed business domain, so the address goes to the clipboard.
    if (page) wx.setClipboardData({ data: `${api.API_URL}/legal/${page}?lang=${i18n.current()}` });
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
    settings.load().catch(() => {});
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
      if (!accepted) throw new Error(t('please accept the terms to continue'));
      this.after(await api.auth.signUp({ email: email.trim(), password, name: name.trim(), accept_terms: true, ref: share.pending(), language: i18n.current() }));
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
        this.after(await api.auth.wechat(await wxLogin(), false, share.pending(), i18n.current()));
      } catch (e) {
        if (e.raw !== 'please accept the terms to continue') throw e;
        if (!(await confirm(t('Welcome to SynAgent'), t('By continuing, you confirm you are 18 or over and accept the Terms of Use and Privacy Policy.')))) return;
        this.after(await api.auth.wechat(await wxLogin(), true, share.pending(), i18n.current()));
      }
    });
  },
  onResend() {
    this.act(async () => {
      await api.auth.resend();
      wx.showToast({ title: t('A new code is on its way.'), icon: 'none' });
    });
  },
  onLeave() {
    api.setToken('');
    this.setData({ mode: 'signin', code: '' });
  },
  onForgot() {
    this.act(async () => {
      await api.auth.forgot(this.data.email.trim());
      this.setData({ mode: 'reset', password: '', code: '', resetLead: this.resetLead() });
    });
  },
  onReset() {
    const { email, code, password } = this.data;
    this.act(async () => {
      await api.auth.reset(email.trim(), code.trim(), password);
      this.setData({ mode: 'signin', password: '', code: '' });
      wx.showToast({ title: t('Password changed. Sign in with the new one.'), icon: 'none' });
    });
  },
});
