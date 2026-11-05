const api = require('../../lib/api');
const live = require('../../lib/live');
const i18n = require('../../lib/i18n');
const person = require('../../lib/person');
const { showError, confirm } = require('../../lib/format');

const { t } = i18n;

const COPY = {
  aiGenerated: 'AI-generated',
  plan: 'Plan a date',
  more: 'More',
  ended: "This chat has ended. I'm sorry. There are more people to meet.",
  send: 'Send',
  ask: 'Ask Suki',
  justToYou: 'Suki, just to you',
  onlyYou: 'Only you can see this.',
  dismiss: 'Dismiss this hint',
  thinking: 'Let me think about this.',
  thinkingShort: 'Suki is thinking about this.',
  noteWaiting: 'Suki left you a note. Close the keyboard to see it.',
  tapSuggestion: 'Tap one to put it in your message box. Nothing sends until you do.',
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
// If her answer gets lost, don't leave "thinking" on screen forever.
const COACH_WAIT_MS = 45000;

Page({
  data: { t: i18n.strings(COPY), id: '', me: '', match: {}, messages: [], draft: '', bottom: '', barText: '', placeholder: '', hint: null, hintCount: 0, asking: false, typing: false, focusInput: false },
  onLoad(q) {
    this.setData({ id: q.id });
    this.off = live.on((topic) => {
      if (topic === 'chat:' + q.id || topic === 'reconnect') {
        this.load();
        this.loadHints();
      } else if (topic === 'matches') this.loadMatch();
    });
    api.auth.me().then((r) => this.setData({ me: r.user.id }), showError);
    this.loadMatch();
  },
  onUnload() {
    if (this.off) this.off();
    clearInterval(this.poll);
    clearTimeout(this.askTimer);
  },
  onShow() {
    i18n.apply(this, COPY, 'Chats');
    this.load();
    this.loadHints();
    clearInterval(this.poll);
    this.poll = setInterval(() => {
      this.load();
      if (this.data.asking) this.loadHints();
    }, POLL_MS);
  },
  onHide() {
    clearInterval(this.poll);
  },
  async loadMatch() {
    try {
      const match = await api.matches.get(this.data.id);
      this.setData({
        match,
        barText: match.activity ? t('Buddy: {activity}', { activity: person.activity(match.activity) }) : (match.person.city_name || match.person.display_name),
        placeholder: t('Message {name}', { name: match.person.display_name }),
      });
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
  async loadHints() {
    try {
      const hints = await api.matches.hints(this.data.id);
      const change = { hint: hints[0] || null, hintCount: hints.length };
      if (hints.length > this.data.hintCount) change.asking = false;
      this.setData(change);
    } catch (_) {
      // The panel just stays as it was.
    }
  },
  async onAsk() {
    if (this.data.asking) return;
    this.setData({ asking: true });
    clearTimeout(this.askTimer);
    this.askTimer = setTimeout(() => this.setData({ asking: false }), COACH_WAIT_MS);
    try {
      await api.matches.coach(this.data.id);
    } catch (e) {
      this.setData({ asking: false });
      showError(e);
    }
  },
  async onDismissHint() {
    const hint = this.data.hint;
    if (!hint) return;
    this.setData({ hint: null, hintCount: Math.max(0, this.data.hintCount - 1) });
    // Any note behind this one comes up next.
    try {
      await api.matches.dismissHint(hint.id);
    } catch (_) {
      // It comes back below.
    }
    await this.loadHints();
  },
  onSuggestion(e) {
    this.setData({ draft: e.currentTarget.dataset.text, focusInput: true });
  },
  // With the keyboard up the panel would cover the whole conversation, so it steps aside.
  onFocus() {
    this.setData({ typing: true });
  },
  onBlur() {
    this.setData({ typing: false, focusInput: false });
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
  onPerson() {
    wx.navigateTo({ url: '/pages/person/person?match=' + this.data.id });
  },
  onMore() {
    wx.showActionSheet({
      itemList: [t('View profile'), t('Report'), t('Block'), t('Unmatch')],
      success: (r) => [this.onPerson, this.report, this.block, this.unmatch][r.tapIndex].call(this),
    });
  },
  report() {
    wx.showActionSheet({
      itemList: REASONS.map((x) => t(x[1])),
      success: async (r) => {
        try {
          await api.safety.report({ user_id: this.data.match.person.user_id, match_id: this.data.id, reason: REASONS[r.tapIndex][0], details: '' });
          wx.showToast({ title: t('Thank you for telling us'), icon: 'none' });
        } catch (e) {
          showError(e);
        }
      },
    });
  },
  async block() {
    if (!(await confirm(t('Block this person?'), t("You'll stop seeing each other, and this match ends right away.")))) return;
    try {
      await api.safety.block(this.data.match.person.user_id);
      wx.navigateBack();
    } catch (e) {
      showError(e);
    }
  },
  async unmatch() {
    if (!(await confirm(t('Unmatch?'), t('The chat ends for both of you, and any open date plan is cancelled.')))) return;
    try {
      await api.matches.unmatch(this.data.id);
      wx.navigateBack();
    } catch (e) {
      showError(e);
    }
  },
});
