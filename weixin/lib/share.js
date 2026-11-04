const api = require('./api');

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
    title: '来和我一起用 SynAgent，Suki 帮你认识值得见的人',
    path: mine ? `/pages/suki/suki?ref=${mine}` : '/pages/suki/suki',
  };
}

module.exports = { remember, pending, clearPending, prepare, message };
