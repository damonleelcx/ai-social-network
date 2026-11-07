// A group plan's chat. Suki's card opens it; everyone in the plan is here.
const api = require('../../lib/api');
const live = require('../../lib/live');
const i18n = require('../../lib/i18n');
const person = require('../../lib/person');
const { showError, confirm } = require('../../lib/format');

const { t } = i18n;

const COPY = {
  more: 'More',
  send: 'Send',
  loading: 'Opening your chat...',
  closed: 'This group chat has closed.',
  gone: "You're not in this group any more, or the plan has ended.",
  placeholder: 'Message the group',
};
const REASONS = [
  ['harassment', 'Harassment'],
  ['fake_profile', 'Fake profile'],
  ['inappropriate_content', 'Inappropriate content'],
  ['scam', 'Scam'],
  ['underage', 'Under 18'],
  ['other', 'Something else'],
];
const POLL_MS = 10000;

function eventLine(m) {
  const name = (m.meta && m.meta.name) || '';
  const event = m.meta && m.meta.event;
  if (event === 'joined') return t('{name} joined.', { name });
  if (event === 'left') return t('{name} left.', { name });
  if (event === 'called_off') return t('{name} called off the plan.', { name });
  return m.body;
}

function groupLine(going, spots) {
  const left = Math.max(0, spots - (going - 1));
  return left > 0 ? t('{n} going · {left} spots left', { n: going, left }) : t('{n} going · full', { n: going });
}

Page({
  data: { t: i18n.strings(COPY), id: '', me: '', request: null, messages: [], draft: '', bottom: '', barText: '', loaded: false, open: true, gone: false },
  onLoad(q) {
    this.setData({ id: q.id });
    this.off = live.on((topic) => {
      if (topic === 'group:' + q.id || topic === 'reconnect') this.load();
    });
    api.auth.me().then((r) => this.setData({ me: r.user.id }), showError);
  },
  onUnload() {
    if (this.off) this.off();
    clearInterval(this.poll);
  },
  onShow() {
    i18n.apply(this, COPY);
    this.load();
    clearInterval(this.poll);
    this.poll = setInterval(() => this.load(), POLL_MS);
  },
  onHide() {
    clearInterval(this.poll);
  },
  async load() {
    const last = this.data.messages[this.data.messages.length - 1];
    try {
      const chat = await api.groups.chat(this.data.id, last ? last.seq : 0);
      const r = chat.request;
      const seen = new Set(this.data.messages.map((m) => m.seq));
      const add = (chat.messages || []).filter((m) => !seen.has(m.seq)).map((m) => Object.assign({}, m, { line: m.role === 'text' ? '' : eventLine(m) }));
      const change = {
        request: r,
        open: chat.open,
        loaded: true,
        barText: [r.place, groupLine(r.going, r.spots)].filter(Boolean).join(' · '),
      };
      if (add.length) {
        change.messages = this.data.messages.concat(add).sort((a, b) => a.seq - b.seq);
        change.bottom = 'end';
      }
      this.setData(change);
      wx.setNavigationBarTitle({ title: person.activity(r.kind) + ' · ' + r.when });
    } catch (e) {
      if (e.status === 404) this.setData({ gone: true, loaded: true });
      else if (e.status !== 401 && e.status !== 403) showError(e);
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
      await api.groups.send(this.data.id, text);
      await this.load();
    } catch (e) {
      this.setData({ draft: text });
      showError(e);
    }
  },
  others() {
    const me = this.data.me;
    return ((this.data.request && this.data.request.members) || []).filter((p) => p.user_id !== me);
  },
  onMore() {
    const r = this.data.request;
    if (!r) return;
    const mine = r.person && r.person.user_id === this.data.me;
    const items = [[t("Who's going"), this.people]];
    if (mine && this.data.open) items.push([t('Call it off'), this.callOff]);
    if (!mine) items.push([t('Leave the group'), this.leave]);
    wx.showActionSheet({
      itemList: items.map((x) => x[0]),
      success: (res) => items[res.tapIndex][1].call(this),
    });
  },
  // Everyone in the group, each with report and block.
  people() {
    const others = this.others();
    if (!others.length) return;
    wx.showActionSheet({
      itemList: others.map((p) => p.display_name),
      success: (res) => {
        const p = others[res.tapIndex];
        wx.showActionSheet({
          itemList: [t('View photos'), t('Report'), t('Block')],
          success: (r2) => [() => api.previewPhotos(p.photo_ids), () => this.report(p), () => this.block(p)][r2.tapIndex](),
        });
      },
    });
  },
  report(p) {
    wx.showActionSheet({
      itemList: REASONS.map((x) => t(x[1])),
      success: async (r) => {
        try {
          await api.safety.report({ user_id: p.user_id, reason: REASONS[r.tapIndex][0], details: 'group chat' });
          wx.showToast({ title: t('Thank you for telling us'), icon: 'none' });
        } catch (e) {
          showError(e);
        }
      },
    });
  },
  async block(p) {
    if (!(await confirm(t('Block this person?'), t("You'll stop seeing each other, and any match ends at once.")))) return;
    try {
      await api.safety.block(p.user_id);
      this.setData({ messages: this.data.messages.filter((m) => m.sender_id !== p.user_id) });
      await this.load();
    } catch (e) {
      showError(e);
    }
  },
  async leave() {
    if (!(await confirm(t('Leave the group?'), t("You'll leave its group chat too.")))) return;
    try {
      await api.buddies.withdraw(this.data.id);
      wx.navigateBack();
    } catch (e) {
      showError(e);
    }
  },
  async callOff() {
    if (!(await confirm(t('Call off this plan?'), t('Everyone who joined is told, and the group chat closes.')))) return;
    try {
      await api.buddies.close(this.data.id);
      await this.load();
    } catch (e) {
      showError(e);
    }
  },
});
