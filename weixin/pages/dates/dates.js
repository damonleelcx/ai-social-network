const api = require('../../lib/api');
const live = require('../../lib/live');
const i18n = require('../../lib/i18n');
const { time, showError, confirm } = require('../../lib/format');

const { t } = i18n;

const COPY = {
  posts: 'Your buddy posts',
  plans: 'Date plans',
  nobody: "Nobody's in yet. I'll let you know when someone is.",
  pick: 'Pick them',
  takeDown: 'Take it down',
  openChat: 'Open chat',
  empty: 'Your date plans and buddy posts show up here. Want company for something? Just tell Suki.',
};
const STATUS = { searching: 'Looking for places', voting: 'Choosing a place', scheduling: 'Finding a time', confirmed: 'On!', completed: 'Finished', cancelled: 'Cancelled' };
const POST = { open: 'Open to join', filled: 'Picked', closed: 'Taken down' };

Page({
  data: { t: i18n.strings(COPY), dates: [], posts: [], loaded: false },
  onLoad() {
    this.off = live.on((topic) => {
      if (topic === 'dates' || topic === 'buddies' || topic === 'reconnect') this.load();
    });
  },
  onUnload() {
    if (this.off) this.off();
  },
  onShow() {
    i18n.apply(this, COPY, 'Dates');
    i18n.tabs();
    this.load();
  },
  async onPullDownRefresh() {
    await this.load();
    wx.stopPullDownRefresh();
  },
  async load() {
    try {
      const [dates, posts] = await Promise.all([api.dates.list(), api.buddies.mine()]);
      this.setData({
        loaded: true,
        dates: dates.map((d) => Object.assign({}, d, { statusText: STATUS[d.status] ? t(STATUS[d.status]) : d.status, whenText: time(d.scheduled_at) })),
        posts: posts.map((p) => Object.assign({}, p, {
          statusText: POST[p.status] ? t(POST[p.status]) : p.status,
          offers: (p.offers || []).map((o) => Object.assign({}, o, { line: t('{name} wants to go too', { name: o.person.display_name }) })),
        })),
      });
    } catch (e) {
      if (e.status !== 401 && e.status !== 403) showError(e);
    }
  },
  onOpen(e) {
    wx.navigateTo({ url: '/pages/date/date?id=' + e.currentTarget.dataset.id });
  },
  onChat(e) {
    wx.navigateTo({ url: '/pages/chat/chat?id=' + e.currentTarget.dataset.id });
  },
  async onPick(e) {
    const { post, user, name } = e.currentTarget.dataset;
    if (!(await confirm(t('Go with {name}?', { name }), t('Everyone else who joined will be told.')))) return;
    try {
      wx.navigateTo({ url: '/pages/chat/chat?id=' + (await api.buddies.pick(post, user)) });
    } catch (err) {
      showError(err);
    }
  },
  async onClose(e) {
    if (!(await confirm(t('Take this post down?'), t("People can't join it any more.")))) return;
    try {
      await api.buddies.close(e.currentTarget.dataset.id);
      await this.load();
    } catch (err) {
      showError(err);
    }
  },
});
