const i18n = require('../../../lib/i18n');
const settings = require('../../../lib/settings');
const { showError } = require('../../../lib/format');

const COPY = {
  loading: 'Getting your settings...',
  language: 'Language',
  languageHint: 'For the app and for Suki.',
  notifyAbout: 'Notify me about',
  matches: 'New introductions and matches',
  messages: 'Messages',
  dates: 'Dates',
  suki: 'Things Suki wants to tell me',
  email: 'Also by email',
  chats: 'Chats',
  receipts: 'Read receipts',
  receiptsHint: "People can see when you've read their message.",
  perDay: 'Introductions per day',
  perDayHint: 'The most Suki will bring you in one day. She may bring fewer.',
};

Page({
  data: { t: i18n.strings(COPY), s: null, languages: i18n.LANGUAGES, daily: [1, 2, 3, 4, 5] },
  onShow() {
    i18n.apply(this, COPY, 'App');
    this.load();
  },
  async load() {
    try {
      this.setData({ s: await settings.load() });
    } catch (e) {
      if (e.status !== 401) showError(e);
    }
  },
  // Shows the change right away, and goes back to what's saved if it doesn't save.
  async save(change) {
    this.setData({ s: Object.assign({}, this.data.s, change) });
    try {
      this.setData({ s: await settings.patch(change) });
    } catch (e) {
      this.setData({ s: settings.current() });
      showError(e);
    }
  },
  onToggle(e) {
    this.save({ [e.currentTarget.dataset.f]: e.detail.value });
  },
  onPick(e) {
    const { f, v } = e.currentTarget.dataset;
    if (this.data.s[f] !== v) this.save({ [f]: v });
  },
});
