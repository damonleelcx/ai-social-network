const i18n = require('./i18n');

const pad = (n) => (n < 10 ? '0' + n : '' + n);

const EN_MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const ES_MONTHS = ['ene', 'feb', 'mar', 'abr', 'may', 'jun', 'jul', 'ago', 'sep', 'oct', 'nov', 'dic'];

// Written out by hand: phones don't all ship full locale data to mini programs.
function time(iso) {
  if (!iso) return '';
  const d = new Date(iso);
  const month = d.getMonth();
  const day = d.getDate();
  const clock = `${pad(d.getHours())}:${pad(d.getMinutes())}`;
  switch (i18n.current()) {
    case 'en':
      return `${EN_MONTHS[month]} ${day}, ${clock}`;
    case 'es':
      return `${day} ${ES_MONTHS[month]}, ${clock}`;
    case 'ko':
      return `${month + 1}월 ${day}일 ${clock}`;
    default:
      return `${month + 1}月${day}日 ${clock}`;
  }
}

function showError(e) {
  wx.showToast({ title: (e && e.message) || i18n.t('Something went wrong. Please try again.'), icon: 'none', duration: 3000 });
}

function confirm(title, content) {
  return new Promise((resolve) => {
    wx.showModal(Object.assign({ title, content: content || '', success: (r) => resolve(!!r.confirm), fail: () => resolve(false) }, i18n.modalButtons()));
  });
}

module.exports = { time, pad, showError, confirm };
