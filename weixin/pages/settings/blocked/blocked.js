const api = require('../../../lib/api');
const i18n = require('../../../lib/i18n');
const { showError, confirm } = require('../../../lib/format');

const { t } = i18n;

const COPY = {
  loading: 'Checking...',
  nobody: 'Nobody blocked',
  nobodyBody: "If anyone ever makes you uncomfortable, block them from the chat menu. They won't be told.",
  unblock: 'Unblock',
};

Page({
  data: { t: i18n.strings(COPY), items: null, error: '' },
  onShow() {
    i18n.apply(this, COPY, 'Blocked people');
    this.load();
  },
  async onPullDownRefresh() {
    await this.load();
    wx.stopPullDownRefresh();
  },
  async load() {
    try {
      this.setData({ items: await api.safety.blocked(), error: '' });
    } catch (e) {
      this.setData({ error: e.message });
    }
  },
  async onUnblock(e) {
    const { id, name } = e.currentTarget.dataset;
    if (!(await confirm(t('Unblock {name}?', { name }), t('Your old match won’t come back, but you could be introduced again.')))) return;
    try {
      await api.safety.unblock(id);
      await this.load();
    } catch (err) {
      showError(err);
    }
  },
});
