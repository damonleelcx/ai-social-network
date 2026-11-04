const pad = (n) => (n < 10 ? '0' + n : '' + n);

function time(iso) {
  if (!iso) return '';
  const d = new Date(iso);
  return `${d.getMonth() + 1}月${d.getDate()}日 ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function showError(e) {
  wx.showToast({ title: (e && e.message) || '出了点问题', icon: 'none', duration: 3000 });
}

function confirm(title, content) {
  return new Promise((resolve) => {
    wx.showModal({ title, content: content || '', success: (r) => resolve(!!r.confirm), fail: () => resolve(false) });
  });
}

module.exports = { time, pad, showError, confirm };
