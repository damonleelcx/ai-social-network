// The unread count on the Me tab, so it is visible from every tab.
const api = require('./api');

const ME = 3;

function badge(n) {
  if (n > 0) wx.setTabBarBadge({ index: ME, text: n > 99 ? '99+' : String(n), fail() {} });
  else wx.removeTabBarBadge({ index: ME, fail() {} });
}

async function refresh() {
  if (!api.getToken()) {
    badge(0);
    return 0;
  }
  try {
    const list = await api.notifications.list();
    const n = list.filter((item) => !item.read).length;
    badge(n);
    return n;
  } catch (e) {
    return 0;
  }
}

module.exports = { refresh, badge };
