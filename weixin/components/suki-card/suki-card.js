// The things Suki puts in the conversation. Same cards as the app, same API.
const api = require('../../lib/api');
const i18n = require('../../lib/i18n');
const person = require('../../lib/person');
const { showError, confirm } = require('../../lib/format');
const pay = require('../../lib/pay');
const share = require('../../lib/share');
const { API_URL } = require('../../config');

// A picture of what a plan is about. It shows no one, so it loads without signing in.
const sceneSrc = (key) => (key ? `${API_URL}/v1/scenes/${key}.jpg` : '');

const { t } = i18n;

const COPY = {
  aiGenerated: 'AI-generated',
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
  buyNotice: pay.NOTICE,
  credits: 'Credits',
  done: 'Done.',
  groupHint: 'A group: people who join are in at once, and you all share a group chat.',
  joinGroup: 'Join the group',
  leaveGroup: 'Leave the group',
  openGroup: 'Open the group chat',
  addTrip: 'Add the trip',
  tripHint: "While you're there, plans in this city reach you, and you can post your own.",
  tripGone: 'This trip was removed.',
  recommends: 'Suki recommends',
  sendInvite: 'Send to WeChat friends',
  momentsPoster: 'Share a poster to Moments',
  inviteLine: 'Invite friends to find company with you.',
  lookingTitle: "What I think you're looking for",
  lookingHint: "Only from what you've told me. Tap ✕ to take something out, or tell me what to change.",
  must: 'The basics',
  drawnTo: 'Drawn to',
  steerClear: 'Steering clear of',
  nice: 'Nice to have',
  nothingYet: 'Nothing yet.',
  changeSomething: 'Change something',
  selfieTitle: 'Get the verified badge',
  selfieHint: "One face check: take a selfie now, facing the camera in good light. Our AI provider compares it with your main photo, then it's deleted and only the result is kept. Tapping the button means you agree to this check.",
  takeSelfie: 'Agree and take a selfie',
  verifiedNow: "You're verified.",
  verified: 'Verified',
  wantsToJoin: 'Wants to join you',
};

const GENDERS = { woman: 'Woman', man: 'Man', nonbinary: 'Nonbinary', other: 'Other' };
const FLAGS = { smoking: ['Smokes', person.HABITS], drinking: ['Drinks', person.HABITS], kids: ['Kids', person.KIDS], wants_kids: ['Future kids', person.WANTS_KIDS], religion: ['Religion', null], politics: ['Politics', null] };

// What Suki understands someone wants, as lines of words, from what they told her and nothing else.
function lookingLines(lf) {
  const m = lf.must || {};
  const must = [
    m.seeking && m.seeking.length ? t('Meeting: {who}', { who: m.seeking.map((g) => t(GENDERS[g] || g)).join(', ') }) : '',
    m.age_min && m.age_max ? t('Aged {min} to {max}', { min: m.age_min, max: m.age_max }) : '',
    m.goal ? t('Looking for: {goal}', { goal: t(person.GOALS[m.goal] || m.goal) }) : '',
    m.city ? t('In or near {city}', { city: m.city }) : '',
  ].filter(Boolean);
  const flags = Object.keys(lf.red_flags || {}).map((field) => {
    const [name, options] = FLAGS[field] || [field, null];
    const values = lf.red_flags[field].map((v) => (options && options[v] ? t(options[v]) : v)).join(', ');
    return { field, text: `${t(name)}: ${values}` };
  });
  const read = lf.read || [];
  return { must, flags, nice: lf.nice || [], toward: read.filter((l) => l.kind === 'toward'), away: read.filter((l) => l.kind === 'away') };
}

// Name and age, then work and city, for over a photo.
const heroTitle = (p) => (p ? [p.display_name, p.age ? String(p.age) : ''].filter(Boolean).join(', ') : '');
const heroLine = (p) => (p ? [p.occupation, p.city_name].filter(Boolean).join(' · ') : '');
const CONFIRM = {
  remove_photo: ['Remove this photo?', 'It comes off your profile right away.'],
  unmatch: ['Unmatch?', 'The chat ends for both of you, and any open date plan is cancelled.'],
  block: ['Block this person?', "You'll stop seeing each other, and any match ends at once."],
  reset_taste: ['Start over?', 'I forget my notes and everything I learned about your taste.'],
  buddy_close: ['Take this post down?', "People can't join it any more."],
  trip_remove: ['Remove this trip?', 'Plans there stop reaching you. Plans you posted for it stay up.'],
};

// How full a group is: going counts the poster, spots count only who can join.
function groupLine(going, spots) {
  const left = Math.max(0, spots - (going - 1));
  return left > 0 ? t('{n} going · {left} spots left', { n: going, left }) : t('{n} going · full', { n: going });
}

function day(d) {
  const [, m, dd] = d.split('-').map(Number);
  return m + '/' + dd;
}
const dates = (from, to) => (from === to ? day(from) : day(from) + ' – ' + day(to));

function tripLine(trip) {
  return [
    trip.going === 1 ? t('1 other person is going then') : t('{n} other people are going then', { n: trip.going }),
    trip.plans === 1 ? t('1 plan to join there') : t('{n} plans to join there', { n: trip.plans }),
  ].join(' · ');
}

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
  properties: { card: { type: Object, value: {} }, intro: { type: Boolean, value: false } },
  data: { t: i18n.strings(COPY), packs: [], item: null, photo: '', state: '', error: '', gone: false, offerLine: '', activityText: '', balanceText: '', groupText: '', placeText: '', datesText: '', tripText: '', heroTitle: '', heroLine: '', confirmLabel: '', scene: '', looking: null, askLabel: '' },
  lifetimes: {
    attached() {
      i18n.attach(this, COPY);
      this.load();
    },
    detached() {
      i18n.detach(this);
      clearTimeout(this.sceneTimer);
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
          this.setData({ item, state: item.my_decision || '', heroTitle: heroTitle(item.person), heroLine: heroLine(item.person), askLabel: t('Ask Suki about {name}', { name: item.person.display_name }) });
          this.photo(item.person);
        } else if (c.kind === 'buddy_request' || c.kind === 'group') {
          const item = await api.buddies.get(c.request_id);
          this.setData({
            item,
            activityText: person.activity(item.kind),
            groupText: item.spots > 1 ? groupLine(item.going, item.spots) : '',
            placeText: [item.place, item.city_name].filter(Boolean).join(' · '),
            heroTitle: heroTitle(item.person),
            heroLine: heroLine(item.person),
            scene: sceneSrc(item.scene),
          });
          this.photo(item.person);
        } else if (c.kind === 'trip_post') {
          this.setData({ datesText: dates(c.starts_on, c.ends_on), scene: sceneSrc(c.scene) });
        } else if (c.kind === 'trip') {
          const item = await api.trips.get(c.trip_id);
          this.setData({ item, datesText: dates(item.starts_on, item.ends_on), tripText: tripLine(item), scene: sceneSrc(item.scene) });
        } else if (c.kind === 'buddy_post') {
          this.setData({
            activityText: person.activity(c.activity),
            groupText: c.spots > 1 ? groupLine(1, c.spots) : '',
            placeText: [c.place, c.city_name].filter(Boolean).join(' · '),
            scene: sceneSrc(c.scene),
          });
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
            heroTitle: heroTitle(offer && offer.person),
            heroLine: heroLine(offer && offer.person),
          });
          if (offer) this.photo(offer.person);
        } else if (c.kind === 'looking_for') {
          this.setData({ looking: lookingLines(await api.lookingFor.get()) });
        } else if (c.kind === 'confirm') {
          this.setData({ confirmLabel: t(c.label) });
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
    // A new picture takes a while to draw and isn't there until then, so keep asking.
    onSceneError() {
      this.sceneTries = (this.sceneTries || 0) + 1;
      if (this.sceneTries > 40) return;
      this.sceneTimer = setTimeout(() => {
        this.setData({ scene: `${this.data.scene.split('?')[0]}?try=${this.sceneTries}` });
      }, 3000);
    },
    onPreview() {
      const person = this.data.item && this.data.item.person;
      if (person) api.previewPhotos(person.photo_ids);
    },
    // Asks Suki in the chat, which is where everything about a person is talked through.
    onAsk() {
      const name = this.data.item && this.data.item.person ? this.data.item.person.display_name : '';
      this.triggerEvent('ask', { text: t('Tell me more about {name}. What should I know before I decide?', { name }) }, { bubbles: true, composed: true });
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
      this.run('done', () =>
        api.buddies.post({
          kind: c.activity,
          when: c.when,
          note: c.note || '',
          expires_in_hours: c.expires_in_hours,
          same_gender: !!c.same_gender,
          spots: c.spots,
          place: c.place,
          trip_id: c.trip_id,
        }),
      );
    },
    async onOffer() {
      const id = this.data.card.request_id;
      const offered = this.data.item.offered;
      const group = this.data.item.spots > 1;
      if (offered && group && !(await confirm(t('Leave the group?'), t("You'll leave its group chat too.")))) return;
      this.run('', async () => {
        await (offered ? api.buddies.withdraw(id) : api.buddies.offer(id));
        if (group) {
          const item = await api.buddies.get(id);
          this.setData({ item, groupText: groupLine(item.going, item.spots) });
          if (!offered) this.onOpenGroup();
        } else this.setData({ 'item.offered': !offered });
      });
    },
    async onDropFlag(e) {
      try {
        await api.lookingFor.dropRedFlag(e.currentTarget.dataset.field);
        this.setData({ looking: lookingLines(await api.lookingFor.get()) });
      } catch (err) {
        showError(err);
      }
    },
    async onDropNote(e) {
      try {
        await api.memories.remove(e.currentTarget.dataset.id);
        this.setData({ looking: lookingLines(await api.lookingFor.get()) });
      } catch (err) {
        showError(err);
      }
    },
    async onDropRead(e) {
      try {
        await api.lookingFor.dropRead(e.currentTarget.dataset.id);
        this.setData({ looking: lookingLines(await api.lookingFor.get()) });
      } catch (err) {
        showError(err);
      }
    },
    onChangeLooking() {
      this.triggerEvent('ask', { text: t("I'd like to change what I'm looking for.") }, { bubbles: true, composed: true });
    },
    onSelfie() {
      this.run('', async () => {
        const r = await api.verifySelfie();
        if (!r) return;
        if (r.verified) this.setData({ state: 'done' });
        else this.setData({ error: t(r.reason || "I couldn't be sure it's you. Try again facing the camera in good light, without sunglasses or a mask.") });
      });
    },
    async onPoster() {
      try {
        await share.poster();
      } catch (e) {
        showError(e);
      }
    },
    onOpenGroup() {
      wx.navigateTo({ url: '/pages/group/group?id=' + this.data.card.request_id });
    },
    onAddTrip() {
      const c = this.data.card;
      this.run('done', async () => {
        const item = await api.trips.add({ city: c.city, starts_on: c.starts_on, ends_on: c.ends_on, note: c.note || '' });
        this.setData({ item, tripText: tripLine(item) });
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
        else if (c.action === 'trip_remove') await api.trips.remove(c.trip_id);
      });
    },
  },
});
