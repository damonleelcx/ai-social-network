const api = require('../../lib/api');
const i18n = require('../../lib/i18n');
const person = require('../../lib/person');
const { showError, confirm } = require('../../lib/format');

const { t } = i18n;

const COPY = {
  loading: 'Looking nearby...',
  safety: 'For a first meet-up, pick somewhere public and tell a friend. Never send money to someone you met here.',
  missingTitle: 'Finish your profile first',
  missingBody: "Add your name, birthday, city and a photo first, so people know who's coming.",
  emptyTitle: 'Nothing nearby yet',
  emptyBody: 'When someone nearby posts a plan, it shows up here.',
  join: "I'm in",
  leave: 'Take back',
  joinGroup: 'Join the group',
  leaveGroup: 'Leave the group',
  chat: 'Open the group chat',
  waiting: "{name} will see you're in, and picks who to go with.",
};

Page({
  data: { t: i18n.strings(COPY), loaded: false, posts: [], missing: [] },
  onShow() {
    i18n.apply(this, COPY, 'Nearby plans');
    this.load();
  },
  async onPullDownRefresh() {
    await this.load();
    wx.stopPullDownRefresh();
  },
  async load() {
    if (!api.getToken()) {
      wx.navigateTo({ url: '/pages/login/login' });
      return;
    }
    try {
      const board = await api.buddies.board();
      const posts = (board.requests || []).map((p) => Object.assign({}, p, {
        kindText: person.activity(p.kind),
        who: (p.person && p.person.display_name) || '',
        where: [p.place, p.city_name].filter(Boolean).join(' · '),
        group: p.spots > 1,
        joinLabel: p.spots > 1 ? (p.offered ? t(COPY.leaveGroup) : t(COPY.joinGroup)) : (p.offered ? t(COPY.leave) : t(COPY.join)),
        waiting: p.offered && p.spots <= 1 && p.person ? t(COPY.waiting, { name: p.person.display_name }) : '',
      }));
      this.setData({ loaded: true, posts, missing: board.missing || [] });
      posts.forEach(async (p, i) => {
        const id = p.person && p.person.photo_ids && p.person.photo_ids[0];
        if (id) this.setData({ [`posts[${i}].photo`]: await api.media(id) });
      });
    } catch (e) {
      this.setData({ loaded: true });
      if (e.status !== 401 && e.status !== 403) showError(e);
    }
  },
  async onJoin(e) {
    const post = this.data.posts[e.currentTarget.dataset.index];
    if (!post) return;
    if (!post.offered) {
      const title = post.group ? t("Join {name}'s group?", { name: post.who }) : t('Join {name}?', { name: post.who });
      if (!(await confirm(title, t(COPY.safety)))) return;
    } else if (post.group) {
      if (!(await confirm(t('Leave the group?'), t("You'll leave its group chat too.")))) return;
    }
    try {
      if (post.offered) await api.buddies.withdraw(post.id);
      else await api.buddies.offer(post.id);
      await this.load();
    } catch (err) {
      showError(err);
    }
  },
  onChat(e) {
    wx.navigateTo({ url: '/pages/group/group?id=' + e.currentTarget.dataset.id });
  },
});
