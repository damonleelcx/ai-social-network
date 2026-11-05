const api = require('../../../lib/api');
const live = require('../../../lib/live');
const i18n = require('../../../lib/i18n');
const { time, showError, confirm } = require('../../../lib/format');

const { t } = i18n;

const COPY = {
  signedInAs: 'Signed in as',
  wechatAccount: 'WeChat account',
  changePassword: 'Change password',
  currentPassword: 'Current password',
  newPassword: 'New password',
  atLeast: 'At least 10 characters.',
  changed: 'Changed. Your other devices were signed out.',
  sessions: 'Active sessions',
  thisPhone: 'This phone',
  revoke: 'Revoke',
  yourData: 'Your data',
  dataHint: "A JSON file of your profile, your words and Suki's notes.",
  exportData: 'Export my data',
  sendFile: 'Send the file to a WeChat chat',
  legal: 'Legal',
  legalHint: 'Tapping one copies its link, so you can open it in a browser.',
  terms: 'Terms',
  privacy: 'Privacy policy',
  contact: 'Contact',
  howDeletion: 'How account deletion works',
  signOut: 'Sign out',
  deleteAccount: 'Delete my account',
  deleteTitle: 'Delete your account',
  deleteWithPassword: "This removes your profile, photos, matches and chats for good. It can't be undone. Type your password to confirm.",
  deleteWithWeChat: "This removes your profile, photos, matches and chats for good. It can't be undone.",
  confirmWeChat: 'Confirm with WeChat to finish.',
  password: 'Password',
  keep: 'Keep my account',
};

function describeAgent(ua) {
  if (!ua) return t('Unknown device');
  if (/okhttp|android/i.test(ua)) return t('Android phone');
  if (/darwin|cfnetwork|iphone|ios/i.test(ua)) return t('iPhone');
  return ua.length > 40 ? `${ua.slice(0, 40)}...` : ua;
}

const wxLogin = () => new Promise((resolve, reject) => {
  wx.login({ success: (r) => resolve(r.code), fail: () => reject(new Error(t("WeChat sign-in didn't work. Please try again."))) });
});

Page({
  data: {
    t: i18n.strings(COPY),
    user: null,
    sessions: [],
    sessionError: '',
    passwordOpen: false,
    current: '',
    next: '',
    passwordBusy: false,
    passwordError: '',
    passwordDone: false,
    exporting: false,
    exportPath: '',
    deleteOpen: false,
    deletePassword: '',
    deleteBusy: false,
    deleteError: '',
    signingOut: false,
  },
  onShow() {
    i18n.apply(this, COPY, 'Account');
    this.load();
  },
  onUnload() {
    this.removeExport();
  },
  async onPullDownRefresh() {
    await this.load();
    wx.stopPullDownRefresh();
  },
  async load() {
    try {
      const me = await api.auth.me();
      this.setData({ user: me.user });
    } catch (e) {
      if (e.status !== 401) showError(e);
    }
    this.loadSessions();
  },
  async loadSessions() {
    try {
      const sessions = await api.auth.sessions();
      this.setData({
        sessionError: '',
        sessions: sessions.map((s) => ({
          id: s.id,
          current: s.current,
          device: describeAgent(s.user_agent),
          seen: t('Last seen {time} {value}', { time: time(s.last_seen_at), value: s.ip ? `· ${s.ip}` : '' }),
        })),
      });
    } catch (e) {
      this.setData({ sessionError: e.message });
    }
  },
  onInput(e) {
    this.setData({ [e.currentTarget.dataset.f]: e.detail.value });
  },
  onPasswordOpen() {
    this.setData({ passwordOpen: !this.data.passwordOpen, current: '', next: '', passwordError: '', passwordDone: false });
  },
  async onPasswordSave() {
    const { current, next } = this.data;
    this.setData({ passwordBusy: true, passwordError: '' });
    try {
      await api.auth.changePassword(current, next);
      this.setData({ passwordDone: true, current: '', next: '' });
      this.loadSessions();
    } catch (e) {
      this.setData({ passwordError: e.message });
    }
    this.setData({ passwordBusy: false });
  },
  async onRevoke(e) {
    const { id, current } = e.currentTarget.dataset;
    const body = current ? t('This signs you out on this phone.') : t('That device gets signed out right away.');
    if (!(await confirm(t('Sign out this session?'), body))) return;
    try {
      await api.auth.revokeSession(id);
      if (current) this.leave();
      else await this.loadSessions();
    } catch (err) {
      showError(err);
    }
  },
  // WeChat only sends a file from a tap, so this fetches it, then offers a second button.
  async onExport() {
    this.setData({ exporting: true });
    try {
      const data = await api.auth.exportData();
      const filePath = `${wx.env.USER_DATA_PATH}/synagent-data.json`;
      await new Promise((resolve, reject) => {
        wx.getFileSystemManager().writeFile({ filePath, data: JSON.stringify(data, null, 2), encoding: 'utf8', success: resolve, fail: reject });
      });
      this.setData({ exportPath: filePath });
    } catch (e) {
      showError(e);
    }
    this.setData({ exporting: false });
  },
  onSendFile() {
    wx.shareFileMessage({
      filePath: this.data.exportPath,
      fileName: 'synagent-data.json',
      fail: (e) => {
        if (!/cancel/.test((e && e.errMsg) || '')) showError(new Error(t('Something went wrong. Please try again.')));
      },
    });
  },
  // It's the person's own data, so it doesn't stay on the phone after this page.
  removeExport() {
    if (!this.data.exportPath) return;
    wx.getFileSystemManager().unlink({ filePath: this.data.exportPath, fail: () => {} });
  },
  onLegal(e) {
    // A web-view needs a filed business domain, so the address goes to the clipboard.
    wx.setClipboardData({ data: `${api.API_URL}/legal/${e.currentTarget.dataset.page}?lang=${i18n.current()}` });
  },
  leave() {
    api.setToken('');
    live.disconnect();
    wx.reLaunch({ url: '/pages/login/login' });
  },
  async onSignOut() {
    this.setData({ signingOut: true });
    try {
      await api.auth.signOut();
    } catch (_) {
      // Signing out here still happens.
    }
    this.leave();
  },
  onDeleteOpen() {
    this.setData({ deleteOpen: !this.data.deleteOpen, deletePassword: '', deleteError: '' });
  },
  async onDelete() {
    this.setData({ deleteBusy: true, deleteError: '' });
    try {
      if (this.data.user.has_password) await api.auth.deleteAccount(this.data.deletePassword);
      else await api.auth.deleteWithWeChat(await wxLogin());
      this.leave();
    } catch (e) {
      this.setData({ deleteError: e.message, deleteBusy: false });
    }
  },
});
