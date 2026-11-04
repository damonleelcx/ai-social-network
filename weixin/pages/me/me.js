const api = require('../../lib/api');
const live = require('../../lib/live');
const { time, showError, confirm } = require('../../lib/format');
const pay = require('../../lib/pay');
const share = require('../../lib/share');

const REASON = { purchase: '购买积分', turn: '发给 Suki 的消息', referral: '朋友加入', refund: '退回：Suki 没能回复', grant: 'SynAgent 赠送' };

Page({
  data: { user: {}, photos: [], invite: null, credits: null, usage: [], packs: [], buying: false },
  onLoad() {
    this.off = live.on((t) => {
      if (t === 'credits' || t === 'photos' || t === 'reconnect') this.load();
    });
  },
  onUnload() {
    if (this.off) this.off();
  },
  onShow() {
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
      const [me, photos, credits, usage] = await Promise.all([api.auth.me(), api.photos.list(), api.credits.status(), api.credits.usage()]);
      this.setData({
        user: me.user,
        credits,
        packs: pay.packs(),
        usage: usage.slice(0, 20).map((u) => ({ delta: u.delta, text: REASON[u.reason] || u.reason, when: time(u.created_at) })),
      });
      this.setData({ invite: await share.prepare() });
      const shown = await Promise.all(photos.map(async (p) => ({ id: p.id, path: await api.media(p.id) })));
      this.setData({ photos: shown });
    } catch (e) {
      if (e.status !== 401 && e.status !== 403) showError(e);
    }
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
        wx.showToast({ title: '收到，正在审核', icon: 'none' });
        await this.load();
      }
    } catch (e) {
      showError(e);
    }
  },
  onPhoto(e) {
    const id = e.currentTarget.dataset.id;
    wx.showActionSheet({
      itemList: ['设为头像', '删除'],
      success: async (r) => {
        try {
          if (r.tapIndex === 0) await api.photos.setPrimary(id);
          else if (await confirm('删除这张照片？', '会立刻从你的资料里移除。')) await api.photos.remove(id);
          await this.load();
        } catch (err) {
          showError(err);
        }
      },
    });
  },
  onLegal(e) {
    wx.setClipboardData({ data: `${api.API_URL}/legal/${e.currentTarget.dataset.page}` });
  },
  leave() {
    api.setToken('');
    live.disconnect();
    wx.reLaunch({ url: '/pages/login/login' });
  },
  async onSignOut() {
    try {
      await api.auth.signOut();
    } catch (_) {
      // Signing out here still happens.
    }
    this.leave();
  },
  async onDelete() {
    if (!(await confirm('删除账号？', '你的资料、匹配和聊天都会删除，无法恢复。'))) return;
    if (!this.data.user.has_password) {
      wx.login({
        success: async (r) => {
          try {
            await api.auth.deleteWithWeChat(r.code);
            this.leave();
          } catch (e) {
            showError(e);
          }
        },
        fail: () => showError(new Error('微信登录失败，请再试一次。')),
      });
      return;
    }
    wx.showModal({
      title: '输入密码确认',
      editable: true,
      placeholderText: '密码',
      success: async (r) => {
        if (!r.confirm || !r.content) return;
        try {
          await api.auth.deleteAccount(r.content);
          this.leave();
        } catch (e) {
          showError(e);
        }
      },
    });
  },
});
