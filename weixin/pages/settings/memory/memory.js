const api = require('../../../lib/api');
const i18n = require('../../../lib/i18n');
const settings = require('../../../lib/settings');
const { showError, confirm } = require('../../../lib/format');

const { t } = i18n;

const COPY = {
  off: "Long-term memory is off, so I'm not adding anything new.",
  loading: 'Going through my notes...',
  blank: 'A blank page',
  blankBody: "I haven't written anything down yet. As we talk, my notes will show up here, and you can delete any of them.",
  remove: 'Remove',
  reset: 'Forget everything and start over',
};

Page({
  data: { t: i18n.strings(COPY), items: null, memoryOff: false, resetting: false, error: '' },
  onShow() {
    i18n.apply(this, COPY, 'What Suki remembers');
    this.load();
  },
  async onPullDownRefresh() {
    await this.load();
    wx.stopPullDownRefresh();
  },
  async load() {
    try {
      const [items, s] = await Promise.all([api.memories.list(), settings.load()]);
      this.setData({ items, memoryOff: !s.memory_enabled, error: '' });
    } catch (e) {
      this.setData({ error: e.message });
    }
  },
  async onRemove(e) {
    const id = e.currentTarget.dataset.id;
    this.setData({ items: this.data.items.filter((m) => m.id !== id) });
    try {
      await api.memories.remove(id);
    } catch (err) {
      showError(err);
      this.load();
    }
  },
  async onReset() {
    if (!(await confirm(t('Start over?'), t("I'll forget my notes about you and everything I've learned about your taste. This can't be undone.")))) return;
    this.setData({ resetting: true });
    try {
      await api.memories.reset();
      await this.load();
    } catch (e) {
      showError(e);
    }
    this.setData({ resetting: false });
  },
});
