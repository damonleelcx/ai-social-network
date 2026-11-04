const api = require('../../lib/api');
const live = require('../../lib/live');
const { time, pad, showError, confirm } = require('../../lib/format');

Page({
  data: { id: '', date: null, day: '', clock: '', today: '', proposedText: '', scheduledText: '' },
  onLoad(q) {
    const n = new Date();
    this.setData({ id: q.id, today: `${n.getFullYear()}-${pad(n.getMonth() + 1)}-${pad(n.getDate())}` });
    this.off = live.on((t) => {
      if (t === 'dates' || t === 'reconnect') this.load();
    });
    this.load();
  },
  onUnload() {
    if (this.off) this.off();
  },
  show(date) {
    this.setData({ date, proposedText: time(date.proposed_at), scheduledText: time(date.scheduled_at) });
  },
  async load() {
    try {
      this.show(await api.dates.get(this.data.id));
    } catch (e) {
      showError(e);
    }
  },
  async act(fn) {
    try {
      this.show(await fn());
    } catch (e) {
      showError(e);
    }
  },
  onVote(e) {
    this.act(() => api.dates.vote(this.data.id, e.currentTarget.dataset.id));
  },
  onDay(e) {
    this.setData({ day: e.detail.value });
  },
  onClock(e) {
    this.setData({ clock: e.detail.value });
  },
  onTime() {
    const [y, m, d] = this.data.day.split('-').map(Number);
    const [h, min] = this.data.clock.split(':').map(Number);
    this.act(() => api.dates.time(this.data.id, new Date(y, m - 1, d, h, min).toISOString()));
  },
  onConfirm() {
    this.act(() => api.dates.confirm(this.data.id));
  },
  async onCancel() {
    if (await confirm('取消这次约会？', '对方会收到通知。')) this.act(() => api.dates.cancel(this.data.id));
  },
  async onShare() {
    try {
      wx.setClipboardData({ data: await api.dates.share(this.data.id) });
    } catch (e) {
      showError(e);
    }
  },
});
