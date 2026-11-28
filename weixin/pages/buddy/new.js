const api = require('../../lib/api');
const i18n = require('../../lib/i18n');
const person = require('../../lib/person');
const { showError } = require('../../lib/format');

const { t } = i18n;

const COPY = {
  ideaTitle: 'What do you want to do?',
  ideaBody: 'One sentence is enough. I can fill in the rest, and nothing is posted until you say so.',
  idea: 'Your idea',
  ideaPlaceholder: 'Ramen this Friday, two people',
  fill: 'Fill it in for me',
  kind: 'Kind',
  when: 'When',
  whenPlaceholder: 'Saturday evening',
  note: 'Note',
  notePlaceholder: 'What you want to do',
  place: 'Place',
  placePlaceholder: 'Optional',
  hours: 'How long it stays up',
  spots: 'How many can join',
  same: 'Only people of my gender',
  post: 'Post it',
  needIdea: 'Say what you want to do, in a sentence.',
  needWhen: 'Say when, in a few words.',
};

const KINDS = ['movie', 'meal', 'show', 'travel', 'sports', 'games', 'business', 'study', 'other'];
const HOURS = [
  { value: 12, label: 'About tonight' },
  { value: 24, label: 'A day' },
  { value: 72, label: 'A few days' },
  { value: 168, label: 'A week' },
  { value: 336, label: 'Two weeks' },
];
const SPOTS = [1, 2, 3, 5, 8];

function nearest(list, n) {
  return list.reduce((best, v) => (Math.abs(v - n) < Math.abs(best - n) ? v : best), list[0]);
}

Page({
  data: {
    t: i18n.strings(COPY),
    idea: '',
    kind: 'meal',
    when: '',
    note: '',
    place: '',
    hours: 24,
    spots: 1,
    same: false,
    kinds: [],
    hourOptions: [],
    spotOptions: [],
    drafting: false,
    posting: false,
  },
  onShow() {
    i18n.apply(this, COPY, 'Post a plan');
    this.setData({
      kinds: KINDS.map((value) => ({ value, label: person.activity(value) })),
      hourOptions: HOURS.map((h) => ({ value: h.value, label: t(h.label) })),
      spotOptions: SPOTS.map((value) => ({ value, label: String(value) })),
    });
  },
  onIdea(e) { this.setData({ idea: e.detail.value }); },
  onWhen(e) { this.setData({ when: e.detail.value }); },
  onNote(e) { this.setData({ note: e.detail.value }); },
  onPlace(e) { this.setData({ place: e.detail.value }); },
  onKind(e) { this.setData({ kind: e.currentTarget.dataset.value }); },
  onHours(e) { this.setData({ hours: Number(e.currentTarget.dataset.value) }); },
  onSpots(e) { this.setData({ spots: Number(e.currentTarget.dataset.value) }); },
  onSame(e) { this.setData({ same: e.detail.value }); },
  async onFill() {
    const idea = this.data.idea.trim();
    if (!idea) return showError({ message: t(COPY.needIdea) });
    this.setData({ drafting: true });
    try {
      const draft = await api.buddies.draft(idea);
      const hours = nearest(HOURS.map((h) => h.value), draft.expires_in_hours || 24);
      const spots = nearest(SPOTS, draft.spots || 1);
      this.setData({
        kind: KINDS.includes(draft.kind) ? draft.kind : 'other',
        when: draft.when || this.data.when,
        note: draft.note || this.data.note,
        place: draft.place || this.data.place,
        hours,
        spots,
        same: !!draft.same_gender,
      });
    } catch (e) {
      showError({ message: t((e && e.message) || "I couldn't draft that just now. You can fill the form yourself.") });
    } finally {
      this.setData({ drafting: false });
    }
  },
  async onPost() {
    if (!this.data.when.trim()) return showError({ message: t(COPY.needWhen) });
    this.setData({ posting: true });
    try {
      await api.buddies.post({
        kind: this.data.kind,
        when: this.data.when.trim(),
        note: this.data.note.trim(),
        place: this.data.place.trim(),
        expires_in_hours: this.data.hours,
        same_gender: this.data.same,
        spots: this.data.spots,
      });
      wx.navigateBack();
    } catch (e) {
      showError({ message: t((e && e.message) || 'Something went wrong. Please try again.') });
      this.setData({ posting: false });
    }
  },
});
