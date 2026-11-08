const api = require('./api');
const i18n = require('./i18n');
const { API_URL } = require('../config');

const { t } = i18n;

const PENDING = 'pending_ref';
let mine = '';

// The code in the link someone opened. It waits here until they sign up.
function remember(query) {
  if (query && query.ref) wx.setStorageSync(PENDING, query.ref);
}

function pending() {
  return wx.getStorageSync(PENDING) || '';
}

function clearPending() {
  wx.removeStorageSync(PENDING);
}

// Fetched ahead of time because the share sheet asks for its message right away.
async function prepare() {
  if (!api.getToken()) {
    mine = '';
    return null;
  }
  try {
    const st = await api.referral.status();
    mine = st.code;
    return st;
  } catch (e) {
    return null;
  }
}

// The message for onShareAppMessage. It doesn't mention a reward: WeChat's rules don't
// allow offering one for sharing.
function message() {
  return {
    title: t('Come try SynAgent with me. Suki introduces you to people worth meeting.'),
    path: mine ? `/pages/suki/suki?ref=${mine}` : '/pages/suki/suki',
  };
}

// For Moments (朋友圈). It opens the Suki page on its own, so the code goes in the query.
function timeline() {
  return {
    title: t('Come try SynAgent with me. Suki introduces you to people worth meeting.'),
    query: mine ? `ref=${mine}` : '',
  };
}

// This person's poster, with their QR code and invite code, offered through WeChat's own image
// menu: send to a friend, share to Moments, or save. Nothing about a reward is on it.
function poster() {
  return new Promise((resolve, reject) => {
    wx.showLoading({ title: t('One moment...'), mask: true });
    wx.downloadFile({
      url: `${API_URL}/v1/referral/poster?lang=${i18n.current()}`,
      header: api.headers(),
      success: (r) => {
        wx.hideLoading();
        if (r.statusCode !== 200) return reject(new Error(t("The poster didn't load. Please try again.")));
        if (!wx.showShareImageMenu) {
          wx.previewImage({ urls: [r.tempFilePath] });
          return resolve();
        }
        wx.showShareImageMenu({ path: r.tempFilePath, success: () => resolve(), fail: () => resolve() });
      },
      fail: () => {
        wx.hideLoading();
        reject(new Error(t("The poster didn't load. Please try again.")));
      },
    });
  });
}

module.exports = { remember, pending, clearPending, prepare, message, timeline, poster };
