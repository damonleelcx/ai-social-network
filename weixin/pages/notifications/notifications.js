const api = require('../../lib/api');
const live = require('../../lib/live');
const i18n = require('../../lib/i18n');
const { time, showError } = require('../../lib/format');

const COPY = {
  checking: 'Checking...',
  quiet: 'All quiet',
  quietBody: "When something happens, you'll hear it from me first.",
};

Page({
  data: { t: i18n.strings(COPY), items: [], loaded: false },
  onLoad() {
    // Opening the list reads it, but the ones that were new stay marked for this visit.
    this.fresh = new Set();
    this.off = live.on((topic) => {
      if (topic === 'notification' || topic === 'reconnect') this.load();
    });
  },
  onUnload() {
    if (this.off) this.off();
  },
  onShow() {
    i18n.apply(this, COPY, 'Notifications');
    this.load();
  },
  async onPullDownRefresh() {
    await this.load();
    wx.stopPullDownRefresh();
  },
  async load() {
    try {
      const list = await api.notifications.list();
      const unread = list.filter((n) => !n.read);
      unread.forEach((n) => this.fresh.add(n.id));
      this.setData({
        loaded: true,
        items: list.map((n) => ({ id: n.id, title: n.title, body: n.body, when: time(n.created_at), data: n.data || {}, fresh: this.fresh.has(n.id) })),
      });
      if (unread.length) api.notifications.markRead().catch(() => {});
    } catch (e) {
      if (e.status !== 401 && e.status !== 403) showError(e);
    }
  },
  onOpen(e) {
    const { screen, id } = e.currentTarget.dataset;
    if (screen === 'agent') wx.switchTab({ url: '/pages/suki/suki' });
    else if (screen === 'proposal' && id) wx.navigateTo({ url: '/pages/person/person?proposal=' + id });
    else if (screen === 'chat' && id) wx.navigateTo({ url: '/pages/chat/chat?id=' + id });
    else if (screen === 'date' && id) wx.navigateTo({ url: '/pages/date/date?id=' + id });
  },
});
