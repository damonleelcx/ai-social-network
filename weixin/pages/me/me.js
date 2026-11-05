const api = require('../../lib/api');
const live = require('../../lib/live');
const i18n = require('../../lib/i18n');
const { time, showError, confirm } = require('../../lib/format');
const pay = require('../../lib/pay');
const share = require('../../lib/share');

const { t } = i18n;

const COPY = {
  editProfile: 'Edit profile',
  editHint: 'Or just tell Suki',
  pause: 'Pause introductions',
  pauseHint: 'You keep your matches and chats. I just stop looking.',
  photos: 'Photos',
  photosHint: 'Tap a photo to see all of it. Use ··· to make it your main one or remove it.',
  invite: 'Invite friends',
  inviteCode: 'Your invite code',
  sendInvite: 'Send to WeChat friends',
  credits: 'Credits',
  noBuying: "Credits can't be bought here right now. You can still use any you already have.",
  more: 'More',
  notifications: 'Notifications',
  settings: 'Settings',
  app: 'App',
  sukiSettings: "Suki's settings",
  sukiSettingsHint: 'How she talks, what she remembers, coaching',
  memory: 'What Suki remembers',
  blocked: 'Blocked people',
  account: 'Account',
  accountHint: 'Password, sessions, your data, legal',
};
const REASON = { purchase: 'Bought credits', turn: 'Message to Suki', referral: 'A friend joined', refund: "Refund: Suki couldn't answer", grant: 'Added by SynAgent' };
const SCREENS = {
  edit: '/pages/profile/edit/edit',
  notifications: '/pages/notifications/notifications',
  app: '/pages/settings/app/app',
  suki: '/pages/settings/suki/suki',
  memory: '/pages/settings/memory/memory',
  blocked: '/pages/settings/blocked/blocked',
  account: '/pages/settings/account/account',
};

Page({
  data: { t: i18n.strings(COPY), user: {}, displayName: '', canPause: false, paused: false, pausing: false, photos: [], invite: null, joinedText: '', credits: null, balanceText: '', creditsText: '', usage: [], packs: [], buying: false, unread: 0 },
  onLoad() {
    this.off = live.on((topic) => {
      if (topic === 'credits' || topic === 'photos' || topic === 'notification' || topic === 'reconnect') this.load();
    });
  },
  onUnload() {
    if (this.off) this.off();
  },
  onShow() {
    i18n.apply(this, COPY, 'Me');
    i18n.tabs();
    if (api.getToken()) this.load();
  },
  onShareAppMessage() {
    return share.message();
  },
  async onPullDownRefresh() {
    await this.load();
    wx.stopPullDownRefresh();
  },
  async load() {
    try {
      const [me, photos, credits, usage, notifications] = await Promise.all([api.auth.me(), api.photos.list(), api.credits.status(), api.credits.usage(), api.notifications.list()]);
      const profile = me.profile || {};
      this.setData({
        user: me.user,
        displayName: profile.display_name || me.user.name,
        canPause: profile.status === 'active' || profile.status === 'paused',
        paused: profile.status === 'paused',
        credits,
        balanceText: t('{n} credits left', { n: credits.balance }),
        creditsText: credits.free
          ? t('Messages to Suki are free right now. You have used {used} of your {free} free match.', { used: Math.min(credits.matches_used, credits.free_matches), free: credits.free_matches })
          : t('Each message to Suki uses {n} credit. Chatting with your matches is free, and so is anything about your safety or your account.', { n: credits.cost_per_message }),
        packs: pay.packs(),
        usage: usage.slice(0, 20).map((u) => ({ delta: u.delta, text: REASON[u.reason] ? t(REASON[u.reason]) : u.reason, when: time(u.created_at) })),
        unread: notifications.filter((n) => !n.read).length,
      });
      const invite = await share.prepare();
      this.setData({ invite, joinedText: invite && invite.joined ? t('{count} people have joined', { count: invite.joined }) : '' });
      const shown = await Promise.all(photos.map(async (p) => ({ id: p.id, status: p.status, is_primary: p.is_primary, path: await api.media(p.id) })));
      this.setData({ photos: shown });
    } catch (e) {
      if (e.status !== 401 && e.status !== 403) showError(e);
    }
  },
  async onPause(e) {
    const paused = e.detail.value;
    this.setData({ paused, pausing: true });
    try {
      await api.profile.pause(paused);
    } catch (err) {
      this.setData({ paused: !paused });
      showError(err);
    }
    this.setData({ pausing: false });
  },
  onOpen(e) {
    wx.navigateTo({ url: SCREENS[e.currentTarget.dataset.screen] });
  },
  async onBuy(e) {
    if (this.data.buying) return;
    this.setData({ buying: true });
    try {
      if (await pay.buy(e.currentTarget.dataset.id)) await this.load();
    } catch (err) {
      showError(err);
    }
    this.setData({ buying: false });
  },
  async onAddPhoto() {
    try {
      if (await api.uploadPhoto()) {
        wx.showToast({ title: t("Got it. I'll take a look."), icon: 'none' });
        await this.load();
      }
    } catch (e) {
      showError(e);
    }
  },
  onPhoto(e) {
    api.previewPhotos(this.data.photos.map((p) => p.id), e.currentTarget.dataset.id);
  },
  onPhotoMore(e) {
    const photo = this.data.photos.find((p) => p.id === e.currentTarget.dataset.id);
    if (!photo) return;
    const canBeMain = photo.status === 'approved' && !photo.is_primary;
    const actions = canBeMain ? ['main', 'remove'] : ['remove'];
    wx.showActionSheet({
      itemList: actions.map((a) => (a === 'main' ? t('Make it my main photo') : t('Remove'))),
      success: async (r) => {
        try {
          if (actions[r.tapIndex] === 'main') await api.photos.setPrimary(photo.id);
          else if (await confirm(t('Remove this photo?'), t('It comes off your profile right away.'))) await api.photos.remove(photo.id);
          await this.load();
        } catch (err) {
          showError(err);
        }
      },
    });
  },
});
