const api = require('../../lib/api');
const live = require('../../lib/live');
const { showError } = require('../../lib/format');

Page({
  data: { matches: [], proposals: [], loaded: false },
  onLoad() {
    this.off = live.on((t) => {
      if (t === 'matches' || t === 'proposals' || t === 'reconnect' || t.startsWith('chat:')) this.load();
    });
  },
  onUnload() {
    if (this.off) this.off();
  },
  onShow() {
    this.load();
  },
  async onPullDownRefresh() {
    await this.load();
    wx.stopPullDownRefresh();
  },
  async load() {
    try {
      const [all, proposals] = await Promise.all([api.matches.list(), api.proposals.list()]);
      const matches = all.filter((m) => m.status === 'active');
      this.setData({
        loaded: true,
        matches,
        proposals: proposals.filter((p) => p.status === 'pending' && !p.my_decision).map((p) => ({ kind: 'proposal', proposal_id: p.id })),
      });
      matches.forEach(async (m, i) => {
        const ids = m.person.photo_ids || [];
        if (ids.length) this.setData({ [`matches[${i}].photo`]: await api.media(ids[0]) });
      });
    } catch (e) {
      if (e.status !== 401 && e.status !== 403) showError(e);
    }
  },
  onOpen(e) {
    wx.navigateTo({ url: '/pages/chat/chat?id=' + e.currentTarget.dataset.id });
  },
});
