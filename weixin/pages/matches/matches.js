const api = require('../../lib/api');
const live = require('../../lib/live');
const i18n = require('../../lib/i18n');
const person = require('../../lib/person');
const { showError } = require('../../lib/format');

const { t } = i18n;

const COPY = {
  introductions: "Suki's introductions",
  chats: 'Chats',
  sayHi: 'Say hi. I know you want to.',
  empty: "No matches yet. Tell Suki who you'd like to meet.",
};

Page({
  data: { t: i18n.strings(COPY), matches: [], proposals: [], loaded: false },
  onLoad() {
    this.off = live.on((topic) => {
      if (topic === 'matches' || topic === 'proposals' || topic === 'reconnect' || topic.startsWith('chat:')) this.load();
    });
  },
  onUnload() {
    if (this.off) this.off();
  },
  onShow() {
    i18n.apply(this, COPY, 'Matches');
    i18n.tabs();
    this.load();
  },
  async onPullDownRefresh() {
    await this.load();
    wx.stopPullDownRefresh();
  },
  async load() {
    try {
      const [all, proposals] = await Promise.all([api.matches.list(), api.proposals.list()]);
      const matches = all.filter((m) => m.status === 'active').map((m) => Object.assign({}, m, {
        activityText: m.activity ? t('Buddy: {activity}', { activity: person.activity(m.activity) }) : '',
      }));
      this.setData({
        loaded: true,
        matches,
        proposals: proposals.filter((p) => p.status === 'open' && !p.my_decision).map((p) => ({ kind: 'proposal', proposal_id: p.id })),
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
  onPhoto(e) {
    const m = this.data.matches[e.currentTarget.dataset.index];
    if (m) api.previewPhotos(m.person.photo_ids);
  },
});
