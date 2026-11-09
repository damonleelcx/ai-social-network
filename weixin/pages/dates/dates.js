const api = require('../../lib/api');
const live = require('../../lib/live');
const i18n = require('../../lib/i18n');
const person = require('../../lib/person');
const { time, showError, confirm } = require('../../lib/format');
const { API_URL } = require('../../config');

const { t } = i18n;

const COPY = {
  posts: 'Your buddy posts',
  plans: 'Date plans',
  nobody: "Nobody's in yet. I'll let you know when someone is.",
  pick: 'Pick them',
  takeDown: 'Take it down',
  openChat: 'Open chat',
  loading: 'Checking your calendar...',
  emptyTitle: 'No dates yet',
  emptyBody: "When a chat feels ready, tap Plan a date, or just ask me. I'll find a spot you'd both like.",
  guestTitle: 'Your plans live here',
  guestBody: 'Meetups and the plans you post show up here once you sign in.',
  signIn: 'Sign in',
};
const STATUS = { searching: 'Looking for places', voting: 'Choosing a place', scheduling: 'Finding a time', confirmed: 'On!', completed: 'Finished', cancelled: 'Cancelled' };
const POST = { open: 'Open to join', filled: 'Picked', closed: 'Taken down' };

Page({
  data: { t: i18n.strings(COPY), guest: false, dates: [], posts: [], loaded: false },
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
    if (!api.getToken()) {
      this.setData({ guest: true, loaded: true });
      return;
    }
    this.load();
  },
  onSignIn() {
    wx.navigateTo({ url: '/pages/login/login' });
  },
  async onPullDownRefresh() {
    if (!this.data.guest) await this.load();
    wx.stopPullDownRefresh();
  },
  async load() {
    if (!api.getToken()) return;
    try {
      const [dates, posts] = await Promise.all([api.dates.list(), api.buddies.mine()]);
      this.setData({
        loaded: true,
        dates: dates.map((d) => Object.assign({}, d, { statusText: STATUS[d.status] ? t(STATUS[d.status]) : d.status, whenText: time(d.scheduled_at) })),
        posts: posts.map((p) => Object.assign({}, p, {
          kindText: person.activity(p.kind),
          sceneUrl: p.scene ? `${API_URL}/v1/scenes/${p.scene}.jpg` : '',
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
