const api = require('../../lib/api');
const live = require('../../lib/live');
const { showError } = require('../../lib/format');

const POLL_MS = 3000;
const POLL_FOR_MS = 180000;

function shape(m) {
  const meta = m.meta || {};
  return { seq: m.seq, role: m.role, body: m.body, cards: meta.cards || [], replies: meta.replies || [] };
}

Page({
  data: { messages: [], thinking: false, draft: '', replies: [], bottom: '', loaded: false },
  onLoad() {
    if (!api.getToken()) {
      wx.reLaunch({ url: '/pages/login/login' });
      return;
    }
    this.off = live.on((topic) => {
      if (topic === 'agent' || topic === 'reconnect') this.load();
      else if (topic === 'typing:agent') this.watch();
    });
  },
  onUnload() {
    if (this.off) this.off();
    this.stopPolling();
  },
  onShow() {
    if (api.getToken()) this.load();
  },
  onHide() {
    this.stopPolling();
  },
  async load() {
    const last = this.data.messages[this.data.messages.length - 1];
    try {
      const r = await api.suki.messages(last ? last.seq : 0);
      const seen = new Set(this.data.messages.map((m) => m.seq));
      const fresh = (r.messages || []).map(shape).filter((m) => !seen.has(m.seq));
      const messages = this.data.messages.concat(fresh);
      const tail = messages[messages.length - 1];
      this.setData({
        messages,
        thinking: r.thinking,
        loaded: true,
        replies: tail && tail.role !== 'user' ? tail.replies : [],
        bottom: fresh.length ? 'end' : this.data.bottom,
      });
      if (r.thinking) this.watch();
      else this.stopPolling();
    } catch (e) {
      if (e.status !== 401 && e.status !== 403) showError(e);
    }
  },
  // The socket says when she's done. This is for when it can't.
  watch() {
    if (this.poll) return;
    const until = Date.now() + POLL_FOR_MS;
    this.poll = setInterval(() => {
      if (Date.now() > until) this.stopPolling();
      else this.load();
    }, POLL_MS);
  },
  stopPolling() {
    clearInterval(this.poll);
    this.poll = null;
  },
  onInput(e) {
    this.setData({ draft: e.detail.value });
  },
  onReply(e) {
    this.send(e.currentTarget.dataset.text);
  },
  onSend() {
    const text = this.data.draft.trim();
    if (!text) return;
    this.setData({ draft: '' });
    this.send(text);
  },
  async send(text) {
    this.setData({ replies: [] });
    try {
      await api.suki.send(text);
      this.setData({ thinking: true });
      this.watch();
      await this.load();
      this.watch();
    } catch (e) {
      this.setData({ draft: text });
      showError(e);
    }
  },
});
