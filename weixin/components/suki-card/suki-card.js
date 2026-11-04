// The things Suki puts in the conversation. Same cards as the app, same API.
const api = require('../../lib/api');
const { showError, confirm } = require('../../lib/format');

const CONFIRM = {
  remove_photo: ['删除这张照片？', '会立刻从你的资料里移除。'],
  unmatch: ['解除匹配？', '聊天会对双方结束，进行中的约会计划也会取消。'],
  block: ['屏蔽这个人？', '你们将不再看到对方，匹配也会立刻结束。'],
  reset_taste: ['重新开始？', '我会忘掉笔记和了解到的你的喜好。'],
  buddy_close: ['撤下这条搭子？', '之后别人就不能报名了。'],
};

const TABS = { matches: '/pages/matches/matches', dates: '/pages/dates/dates' };

Component({
  properties: { card: { type: Object, value: {} } },
  data: { item: null, photo: '', state: '', error: '', gone: false },
  lifetimes: {
    attached() {
      this.load();
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
          this.setData({ item });
          this.photo(item.person);
        } else if (c.kind === 'buddy_offer') {
          const post = await api.buddies.get(c.request_id);
          const offer = (post.offers || []).find((o) => o.person.user_id === c.user_id);
          this.setData({ item: Object.assign({}, post, { person: offer ? offer.person : null }) });
          if (offer) this.photo(offer.person);
        } else if (c.kind === 'buy_credits') {
          this.setData({ item: await api.credits.status() });
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
      wx.switchTab({ url: TABS[screen] || '/pages/me/me' });
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
    async onShare() {
      try {
        wx.setClipboardData({ data: await api.dates.share(this.data.card.date_id) });
      } catch (e) {
        showError(e);
      }
    },
    async onConfirm() {
      const c = this.data.card;
      const [title, body] = CONFIRM[c.action] || ['确定吗？', ''];
      if (!(await confirm(title, body))) return;
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
