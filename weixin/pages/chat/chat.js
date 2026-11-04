const api = require('../../lib/api');
const live = require('../../lib/live');
const { showError, confirm } = require('../../lib/format');

const REASONS = [
  ['harassment', '骚扰'],
  ['fake_profile', '虚假资料'],
  ['inappropriate_content', '不当内容'],
  ['scam', '诈骗'],
  ['underage', '未满 18 岁'],
  ['other', '其他'],
];
const POLL_MS = 10000;

Page({
  data: { id: '', me: '', match: {}, messages: [], draft: '', bottom: '' },
  onLoad(q) {
    this.setData({ id: q.id });
    this.off = live.on((t) => {
      if (t === 'chat:' + q.id || t === 'reconnect') this.load();
      else if (t === 'matches') this.loadMatch();
    });
    api.auth.me().then((r) => this.setData({ me: r.user.id }), showError);
    this.loadMatch();
  },
  onUnload() {
    if (this.off) this.off();
    clearInterval(this.poll);
  },
  onShow() {
    this.load();
    clearInterval(this.poll);
    this.poll = setInterval(() => this.load(), POLL_MS);
  },
  onHide() {
    clearInterval(this.poll);
  },
  async loadMatch() {
    try {
      const match = await api.matches.get(this.data.id);
      this.setData({ match });
      wx.setNavigationBarTitle({ title: match.person.display_name });
    } catch (e) {
      showError(e);
    }
  },
  async load() {
    const last = this.data.messages[this.data.messages.length - 1];
    try {
      const fresh = await api.matches.messages(this.data.id, last ? last.seq : 0);
      const seen = new Set(this.data.messages.map((m) => m.seq));
      const add = fresh.filter((m) => !seen.has(m.seq));
      if (!add.length) return;
      const messages = this.data.messages.concat(add);
      this.setData({ messages, bottom: 'end' });
      api.matches.read(this.data.id, messages[messages.length - 1].seq).catch(() => {});
    } catch (e) {
      if (e.status !== 401 && e.status !== 403) showError(e);
    }
  },
  onInput(e) {
    this.setData({ draft: e.detail.value });
  },
  async onSend() {
    const text = this.data.draft.trim();
    if (!text) return;
    this.setData({ draft: '' });
    try {
      await api.matches.send(this.data.id, text);
      await this.load();
    } catch (e) {
      this.setData({ draft: text });
      showError(e);
    }
  },
  async onDate() {
    try {
      const id = this.data.match.open_date_id || (await api.matches.startDate(this.data.id));
      wx.navigateTo({ url: '/pages/date/date?id=' + id });
    } catch (e) {
      showError(e);
    }
  },
  onMore() {
    wx.showActionSheet({
      itemList: ['举报', '屏蔽', '解除匹配'],
      success: (r) => [this.report, this.block, this.unmatch][r.tapIndex].call(this),
    });
  },
  report() {
    wx.showActionSheet({
      itemList: REASONS.map((x) => x[1]),
      success: async (r) => {
        try {
          await api.safety.report({ user_id: this.data.match.person.user_id, match_id: this.data.id, reason: REASONS[r.tapIndex][0], details: '' });
          wx.showToast({ title: '已收到举报', icon: 'none' });
        } catch (e) {
          showError(e);
        }
      },
    });
  },
  async block() {
    if (!(await confirm('屏蔽这个人？', '你们将不再看到对方，匹配也会立刻结束。'))) return;
    try {
      await api.safety.block(this.data.match.person.user_id);
      wx.navigateBack();
    } catch (e) {
      showError(e);
    }
  },
  async unmatch() {
    if (!(await confirm('解除匹配？', '聊天会对双方结束，进行中的约会计划也会取消。'))) return;
    try {
      await api.matches.unmatch(this.data.id);
      wx.navigateBack();
    } catch (e) {
      showError(e);
    }
  },
});
