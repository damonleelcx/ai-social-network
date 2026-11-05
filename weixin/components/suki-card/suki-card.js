// The things Suki puts in the conversation. Same cards as the app, same API.
const api = require('../../lib/api');
const i18n = require('../../lib/i18n');
const person = require('../../lib/person');
const { showError, confirm } = require('../../lib/format');
const pay = require('../../lib/pay');

const { t } = i18n;

const COPY = {
  gone: 'This one has ended.',
  posted: "Posted. I'll tell you when someone joins.",
  post: 'Post it',
  takeBack: 'Take back',
  imIn: "I'm in",
  picked: 'You picked someone for this one.',
  pick: 'Pick them',
  like: "I'd like to meet",
  pass: 'Pass',
  moment: 'One moment...',
  answered: 'Answered.',
  profile: 'View profile',
  openChat: 'Open chat',
  pickTime: 'Pick a time',
  changePlan: 'Change the plan',
  seePlan: 'See the plan',
  tellFriend: "Tell a friend the plan. It's safer.",
  copyPlan: 'Copy the plan',
  updated: 'Done.',
  useThis: 'Use this',
  photoIn: "Got it. I'll take a look.",
  addPhoto: 'Add a photo',
  noBuying: "Credits can't be bought here right now. You can still use any you already have.",
  done: 'Done.',
};
const CONFIRM = {
  remove_photo: ['Remove this photo?', 'It comes off your profile right away.'],
  unmatch: ['Unmatch?', 'The chat ends for both of you, and any open date plan is cancelled.'],
  block: ['Block this person?', "You'll stop seeing each other, and any match ends at once."],
  reset_taste: ['Start over?', 'I forget my notes and everything I learned about your taste.'],
  buddy_close: ['Take this post down?', "People can't join it any more."],
};

const TABS = { matches: '/pages/matches/matches', dates: '/pages/dates/dates' };
const PAGES = {
  profile_edit: '/pages/profile/edit/edit',
  settings_app: '/pages/settings/app/app',
  settings_agent: '/pages/settings/suki/suki',
  settings_account: '/pages/settings/account/account',
  settings_memory: '/pages/settings/memory/memory',
  settings_blocked: '/pages/settings/blocked/blocked',
};

Component({
  properties: { card: { type: Object, value: {} } },
  data: { t: i18n.strings(COPY), packs: [], item: null, photo: '', state: '', error: '', gone: false, offerLine: '', activityText: '', balanceText: '' },
  lifetimes: {
    attached() {
      i18n.attach(this, COPY);
      this.load();
    },
    detached() {
      i18n.detach(this);
    },
  },
  pageLifetimes: {
    show() {
      i18n.apply(this, COPY);
    },
  },
  methods: {
    async load() {
      const c = this.data.card;
      try {
        if (c.kind === 'proposal') {
          const item = await api.proposals.get(c.proposal_id);
          this.setData({ item, state: item.my_decision || '' });
          this.photo(item.person);
        } else if (c.kind === 'buddy_request') {
          const item = await api.buddies.get(c.request_id);
          this.setData({ item, activityText: person.activity(item.kind) });
          this.photo(item.person);
        } else if (c.kind === 'buddy_post') {
          this.setData({ activityText: person.activity(c.activity) });
          // The card comes back with the conversation, and must not offer to post twice.
          const mine = await api.buddies.mine();
          if (mine.some((p) => p.kind === c.activity && p.when === c.when)) this.setData({ state: 'done' });
        } else if (c.kind === 'buddy_offer') {
          const post = await api.buddies.get(c.request_id);
          const offer = (post.offers || []).find((o) => o.person.user_id === c.user_id);
          this.setData({
            item: Object.assign({}, post, { person: offer ? offer.person : null }),
            activityText: person.activity(post.kind),
            offerLine: offer ? t('{name} wants to go too', { name: offer.person.display_name }) : '',
          });
          if (offer) this.photo(offer.person);
        } else if (c.kind === 'buy_credits') {
          const item = await api.credits.status();
          this.setData({ item, packs: pay.packs(), balanceText: t('{n} credits left', { n: item.balance }) });
        }
      } catch (e) {
        // A post that's over, or that this person can no longer see.
        if (e.status === 404) this.setData({ gone: true });
        else this.setData({ error: e.message });
      }
    },
    async photo(person) {
      const ids = (person && person.photo_ids) || [];
      if (ids.length) this.setData({ photo: await api.media(ids[0]) });
    },
    onPreview() {
      const person = this.data.item && this.data.item.person;
      if (person) api.previewPhotos(person.photo_ids);
    },
    onProfile() {
      wx.navigateTo({ url: '/pages/person/person?proposal=' + this.data.card.proposal_id });
    },
    async run(state, fn) {
      if (this.data.state === 'busy') return;
      const before = this.data.state;
      this.setData({ state: 'busy', error: '' });
      try {
        await fn();
        this.setData({ state });
      } catch (e) {
        this.setData({ state: before, error: e.message });
      }
    },
    openChat(id) {
      wx.navigateTo({ url: '/pages/chat/chat?id=' + id });
    },
    onOpenMatch() {
      this.openChat(this.data.card.match_id);
    },
    onOpenDate() {
      wx.navigateTo({ url: '/pages/date/date?id=' + this.data.card.date_id });
    },
    onOpenScreen() {
      const screen = this.data.card.screen;
      if (screen === 'verify_email') return;
      if (PAGES[screen]) wx.navigateTo({ url: PAGES[screen] });
      else wx.switchTab({ url: TABS[screen] || '/pages/me/me' });
    },
    onDecide(e) {
      const decision = e.currentTarget.dataset.decision;
      this.run(decision, async () => {
        const matchId = await api.proposals.decide(this.data.card.proposal_id, decision);
        if (matchId) this.openChat(matchId);
      });
    },
    onPost() {
      const c = this.data.card;
      this.run('done', () => api.buddies.post({ kind: c.activity, when: c.when, note: c.note || '', expires_in_hours: c.expires_in_hours, same_gender: !!c.same_gender }));
    },
    onOffer() {
      const id = this.data.card.request_id;
      this.run('', async () => {
        const offered = this.data.item.offered;
        await (offered ? api.buddies.withdraw(id) : api.buddies.offer(id));
        this.setData({ 'item.offered': !offered });
      });
    },
    onPick() {
      const c = this.data.card;
      this.run('done', async () => this.openChat(await api.buddies.pick(c.request_id, c.user_id)));
    },
    onBio() {
      this.run('done', () => api.profile.acceptBio());
    },
    onPhoto() {
      this.run('', async () => {
        if (await api.uploadPhoto()) this.setData({ state: 'done' });
      });
    },
    onBuy(e) {
      this.run('', async () => {
        const bought = await pay.buy(e.currentTarget.dataset.id);
        if (bought) this.setData({ item: bought, balanceText: t('{n} credits left', { n: bought.balance }) });
      });
    },
    async onShare() {
      try {
        wx.setClipboardData({ data: await api.dates.share(this.data.card.date_id) });
      } catch (e) {
        showError(e);
      }
    },
    async onConfirm() {
      const c = this.data.card;
      const [title, body] = CONFIRM[c.action] || ['Are you sure?', ''];
      if (!(await confirm(t(title), body ? t(body) : ''))) return;
      this.run('done', async () => {
        if (c.action === 'remove_photo') await api.photos.remove(c.photo_id);
        else if (c.action === 'unmatch') await api.matches.unmatch(c.match_id);
        else if (c.action === 'block') await api.safety.block(c.user_id);
        else if (c.action === 'reset_taste') await api.memories.reset();
        else if (c.action === 'buddy_close') await api.buddies.close(c.request_id);
      });
    },
  },
});
