const api = require('./api');
const { t } = require('./i18n');

const PENDING = 'pending_ref';
// Invite codes are 8 characters from the server's alphabet. Anything else in a link is ignored.
const CODE = /^[A-Za-z0-9]{4,16}$/;
// Without an image WeChat shows a screenshot of the page, which could show someone's chats.
const CARD = '/assets/share/card.jpg';
const SQUARE = '/assets/share/square.jpg';
// Opened from Moments, a page runs on its own with no sign-in (单页模式).
const MOMENTS_SCENE = 1154;
let mine = '';

// The code in the link someone opened. It waits here until they sign up, and only a new
// account is credited to it (the server decides that, not this).
function remember(query) {
  const code = query && query.ref;
  if (code && CODE.test(code)) wx.setStorageSync(PENDING, code);
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

// What a share says. None of it mentions credits or any reward: WeChat forbids offering
// one for sharing (诱导分享), and the reward is for a friend joining, never for the share.
// The link carries only the opaque invite code, never an id or openid.
function message() {
  return {
    title: t('Come try SynAgent with me. Suki introduces you to people worth meeting.'),
    path: mine ? `/pages/suki/suki?ref=${mine}` : '/pages/suki/suki',
    imageUrl: CARD,
  };
}

// For onShareTimeline (Moments). It always opens the page it was shared from, so only pages
// that make sense to a stranger with no account offer it.
function timeline() {
  return {
    title: t('Suki helps you find company for a meal, a film or a trip'),
    query: mine ? `ref=${mine}` : '',
    imageUrl: SQUARE,
  };
}

// True when this launch is the Moments preview, where nobody is signed in and pages can't navigate.
function fromMoments() {
  try {
    return wx.getLaunchOptionsSync().scene === MOMENTS_SCENE;
  } catch (e) {
    return false;
  }
}

module.exports = { remember, pending, clearPending, prepare, message, timeline, fromMoments };
