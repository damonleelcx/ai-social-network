const api = require('../../lib/api');
const live = require('../../lib/live');
const i18n = require('../../lib/i18n');
const { time, pad, showError, confirm } = require('../../lib/format');

const { t } = i18n;

const COPY = {
  loading: 'Getting the plan...',
  cannotOpen: "I can't open this date",
  searchingTitle: 'Looking for places',
  searching: "I'm searching for spots you'd both enjoy. I'll let you know the moment I have a few.",
  tellSuki: 'Tell Suki',
  cancelledBody: 'It happens. You can always plan another one from your chat, if the match is still open.',
  failed: "I couldn't find a good place",
  failedBody: "That's on me. You can try again from your chat, or pick somewhere yourselves.",
  backToChat: 'Back to your chat',
  pickPlace: 'Pick your favorite place.',
  otherVoted: 'They already chose.',
  yourPick: 'Your pick',
  vote: 'Vote for this one',
  youSuggested: 'You suggested',
  thatWorks: 'That works',
  differentTime: 'Suggest a different time',
  suggestTime: 'Pick a date and time',
  day: 'Date',
  time: 'Time',
  choose: 'Choose',
  suggest: 'Suggest this time',
  confirmed: "You're on!",
  copyPlan: 'Copy the plan',
  cancel: 'Cancel this date',
  cancelled: 'This date was called off',
};

Page({
  data: { t: i18n.strings(COPY), id: '', date: null, day: '', clock: '', today: '', proposedText: '', scheduledText: '', heading: '', askHow: '', error: '', otherSuggested: '', waiting: '' },
  onLoad(q) {
    const n = new Date();
    this.setData({ id: q.id, today: `${n.getFullYear()}-${pad(n.getMonth() + 1)}-${pad(n.getDate())}` });
    this.off = live.on((topic) => {
      if (topic === 'dates' || topic === 'reconnect') this.load();
    });
    this.load();
  },
  onUnload() {
    if (this.off) this.off();
  },
  onShow() {
    i18n.apply(this, COPY, 'Date plan');
  },
  show(date) {
    const name = date.person.display_name;
    this.setData({
      date,
      proposedText: time(date.proposed_at),
      scheduledText: time(date.scheduled_at),
      heading: t('A date with {name}', { name }),
      askHow: t('How did it go with {name}?', { name }),
      otherSuggested: t('{name} suggested', { name }),
      waiting: t('Waiting for {name} to say yes.', { name }),
    });
  },
  async load() {
    try {
      this.show(await api.dates.get(this.data.id));
    } catch (e) {
      if (this.data.date) showError(e);
      else this.setData({ error: e.message });
    }
  },
  async act(fn) {
    try {
      this.show(await fn());
    } catch (e) {
      showError(e);
    }
  },
  onTellSuki() {
    wx.switchTab({ url: '/pages/suki/suki' });
  },
  onBackToChat() {
    wx.navigateTo({ url: '/pages/chat/chat?id=' + this.data.date.match_id });
  },
  onPerson() {
    wx.navigateTo({ url: '/pages/person/person?match=' + this.data.date.match_id });
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
    if (await confirm(t('Cancel this date?'), t("They'll be told."))) this.act(() => api.dates.cancel(this.data.id));
  },
  async onShare() {
    try {
      wx.setClipboardData({ data: await api.dates.share(this.data.id) });
    } catch (e) {
      showError(e);
    }
  },
});
