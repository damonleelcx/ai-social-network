// Someone's profile, opened from an introduction (?proposal=) or a match (?match=).
const api = require('../../lib/api');
const live = require('../../lib/live');
const i18n = require('../../lib/i18n');
const person = require('../../lib/person');
const { showError, confirm } = require('../../lib/format');

const { t } = i18n;

const COPY = {
  loading: 'Pulling up their profile...',
  cannotOpen: "I can't open this profile",
  into: 'Into',
  pass: 'Pass',
  yes: 'Yes',
  passed: 'You passed on this one.',
  ended: 'This introduction has ended.',
  translatedBy: 'Translated by Suki',
  ownWords: 'Their own words',
  showOriginal: 'Show original',
  showTranslation: 'Show translation',
};

// What someone wrote about themselves, swapped for the reader's language when there is a translation.
function inLanguage(who, words) {
  if (!words) return who;
  const out = Object.assign({}, who);
  ['bio', 'city_name', 'occupation', 'education', 'interests', 'prompts'].forEach((k) => {
    if (words[k] !== undefined) out[k] = words[k];
  });
  return out;
}

Page({
  data: { t: i18n.strings(COPY), translated: false, original: false, person: null, proposal: null, photos: [], line: '', details: [], open: false, waiting: '', chatLabel: '', chatId: '', busy: '', error: '' },
  onLoad(q) {
    this.q = q;
    this.off = live.on((topic) => {
      if (topic === 'proposals' || topic === 'reconnect') this.load();
    });
    this.load();
  },
  onUnload() {
    if (this.off) this.off();
  },
  onShow() {
    i18n.apply(this, COPY, 'View profile');
  },
  // Their words in this reader's language. Until it arrives, or if it fails, they read as written.
  async translate(who) {
    this.own = who;
    try {
      const r = await api.people.translation(who.user_id, i18n.current());
      if (!r.translated || this.own !== who) return;
      this.words = r.words;
      this.show();
    } catch (_) {
      // Their own words stay.
    }
  },
  show() {
    const who = this.data.original ? this.own : inLanguage(this.own, this.words);
    this.setData({ translated: !!this.words, person: who, line: person.line(who), details: person.details(who) });
  },
  onToggleOriginal() {
    this.setData({ original: !this.data.original });
    this.show();
  },
  async load() {
    try {
      const proposal = this.q.proposal ? await api.proposals.get(this.q.proposal) : null;
      const match = !proposal && this.q.match ? await api.matches.get(this.q.match) : null;
      const who = proposal ? proposal.person : match && match.person;
      if (!who) return;
      const name = who.display_name;
      const chatId = proposal ? proposal.match_id : '';
      this.setData({
        person: who,
        proposal,
        line: person.line(who),
        details: person.details(who),
        open: !!proposal && proposal.status === 'open' && !proposal.my_decision,
        waiting: proposal && proposal.my_decision === 'accept' && !chatId ? t('You said yes. Waiting on {name}.', { name }) : '',
        chatId: chatId || '',
        chatLabel: t('Chat with {name}', { name }),
        error: '',
      });
      wx.setNavigationBarTitle({ title: name });
      this.showPhotos(who.photo_ids || []);
      this.translate(who);
    } catch (e) {
      if (!this.data.person) this.setData({ error: e.message });
      else showError(e);
    }
  },
  async showPhotos(ids) {
    if (ids.join() === this.data.photos.map((p) => p.id).join()) return;
    this.setData({ photos: ids.map((id) => ({ id, path: '' })) });
    ids.forEach(async (id, i) => {
      this.setData({ [`photos[${i}].path`]: await api.media(id) });
    });
  },
  onPhoto(e) {
    api.previewPhotos(this.data.photos.map((p) => p.id), e.currentTarget.dataset.id);
  },
  onChat() {
    wx.redirectTo({ url: '/pages/chat/chat?id=' + this.data.chatId });
  },
  async decide(decision) {
    if (this.data.busy) return;
    this.setData({ busy: decision });
    try {
      const matchId = await api.proposals.decide(this.data.proposal.id, decision);
      if (decision === 'pass') {
        wx.navigateBack();
      } else if (!matchId) {
        await this.load();
      } else {
        await this.load();
        if (await confirm(t('You and {name} both said yes!', { name: this.data.person.display_name }), t("That's a match. Go say hi."))) {
          wx.redirectTo({ url: '/pages/chat/chat?id=' + matchId });
        }
      }
    } catch (e) {
      showError(e);
    }
    this.setData({ busy: '' });
  },
  onPass() {
    this.decide('pass');
  },
  onYes() {
    this.decide('accept');
  },
});
