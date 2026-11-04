const api = require('../../lib/api');
const live = require('../../lib/live');
const { time, showError, confirm } = require('../../lib/format');

const STATUS = { searching: '正在找地方', voting: '选地点', scheduling: '定时间', confirmed: '已确定', completed: '已结束', cancelled: '已取消' };
const POST = { open: '报名中', filled: '已选好', closed: '已撤下' };

Page({
  data: { dates: [], posts: [], loaded: false },
  onLoad() {
    this.off = live.on((t) => {
      if (t === 'dates' || t === 'buddies' || t === 'reconnect') this.load();
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
      const [dates, posts] = await Promise.all([api.dates.list(), api.buddies.mine()]);
      this.setData({
        loaded: true,
        dates: dates.map((d) => Object.assign({}, d, { statusText: STATUS[d.status] || d.status, whenText: time(d.scheduled_at) })),
        posts: posts.map((p) => Object.assign({}, p, { offers: p.offers || [], statusText: POST[p.status] || p.status })),
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
    const { post, user } = e.currentTarget.dataset;
    if (!(await confirm('选 TA 一起去？', '其他报名的人会收到通知。'))) return;
    try {
      wx.navigateTo({ url: '/pages/chat/chat?id=' + (await api.buddies.pick(post, user)) });
    } catch (err) {
      showError(err);
    }
  },
  async onClose(e) {
    if (!(await confirm('撤下这条搭子？', '之后别人就不能报名了。'))) return;
    try {
      await api.buddies.close(e.currentTarget.dataset.id);
      await this.load();
    } catch (err) {
      showError(err);
    }
  },
});
