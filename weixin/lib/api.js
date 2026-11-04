// The server's API, over wx.request. The token lives in this mini program's own storage.
const { API_URL } = require('../config');
const { t } = require('./i18n');
const { showError } = require('./format');

let token = wx.getStorageSync('token') || '';
let leaving = false;

function setToken(next) {
  token = next || '';
  if (token) {
    wx.setStorageSync('token', token);
    return;
  }
  wx.removeStorageSync('token');
  // Her clips and the settings belong to the account that just left.
  require('./voice').clear();
  require('./settings').clear();
}

function leave(url) {
  if (leaving) return;
  leaving = true;
  wx.reLaunch({ url, complete: () => { leaving = false; } });
}

function failure(status, message) {
  if (status === 401 && message === 'please sign in') {
    setToken('');
    require('./live').disconnect();
    leave('/pages/login/login');
  } else if (status === 403 && message === 'please verify your email first') {
    leave('/pages/login/login?mode=verify');
  }
  const err = new Error(t(message || 'Something went wrong on our side. Please try again.'));
  err.status = status;
  err.raw = message || ''; // What the server said, for code that needs to tell errors apart.
  return err;
}

function headers(extra) {
  return Object.assign({ Accept: 'application/json' }, token ? { Authorization: 'Bearer ' + token } : {}, extra);
}

function request(path, method, body, timeout) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: API_URL + path,
      method: method || 'GET',
      data: body,
      timeout: timeout || 20000,
      header: headers(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
      success(res) {
        if (res.statusCode >= 200 && res.statusCode < 300) resolve(res.data);
        else reject(failure(res.statusCode, res.data && res.data.error));
      },
      fail() {
        reject(new Error(t("I can't reach the server right now. Check your connection and try again.")));
      },
    });
  });
}
const get = (p, timeout) => request(p, 'GET', undefined, timeout);
const post = (p, b, timeout) => request(p, 'POST', b === undefined ? {} : b, timeout);
const del = (p) => request(p, 'DELETE');

// PATCH is the one method here that WeChat's wx.request docs don't list, and /v1/settings has
// no other route. If a phone refuses it, the request never gets an HTTP status: that's what
// this catches, so people get a reason instead of being told their network is down.
async function patch(p, b) {
  try {
    return await request(p, 'PATCH', b);
  } catch (e) {
    if (e.status) throw e;
    throw new Error(t("I couldn't save that here. Check your connection, or change it in the SynAgent app."));
  }
}

function clientId() {
  return 'wx-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2, 12);
}

// <image> can't send a token, so photos come down to a temp file first.
const mediaCache = {};
function media(id) {
  if (!id) return Promise.resolve('');
  if (!mediaCache[id]) {
    mediaCache[id] = new Promise((resolve) => {
      wx.downloadFile({
        url: API_URL + '/v1/media/' + id,
        header: headers(),
        success: (r) => resolve(r.statusCode === 200 ? r.tempFilePath : ''),
        fail: () => resolve(''),
      });
    });
    mediaCache[id].then((p) => { if (!p) delete mediaCache[id]; });
  }
  return mediaCache[id];
}

// The whole photo, full screen, with the person's other photos a swipe away. The temp files
// work here for the same reason they work in <image>.
async function previewPhotos(ids, id) {
  const list = (ids || []).filter(Boolean);
  const paths = (await Promise.all(list.map(media))).filter(Boolean);
  const current = await media(id || list[0]);
  if (!current) {
    showError(new Error(t("That photo won't open right now. Please try again.")));
    return;
  }
  wx.previewImage({ urls: paths.length ? paths : [current], current });
}

// The server takes a photo's raw bytes.
function uploadPhoto() {
  return new Promise((resolve, reject) => {
    wx.chooseMedia({
      count: 1,
      mediaType: ['image'],
      sizeType: ['compressed'],
      success(picked) {
        wx.getFileSystemManager().readFile({
          filePath: picked.tempFiles[0].tempFilePath,
          success(file) {
            wx.request({
              url: API_URL + '/v1/photos',
              method: 'POST',
              data: file.data,
              timeout: 60000,
              header: headers({ 'Content-Type': 'image/jpeg' }),
              success(res) {
                if (res.statusCode >= 200 && res.statusCode < 300) resolve(res.data);
                else reject(failure(res.statusCode, res.data && res.data.error));
              },
              fail: () => reject(new Error(t("That photo didn't upload. Please try again."))),
            });
          },
          fail: () => reject(new Error(t("I can't read that photo."))),
        });
      },
      fail: () => resolve(null), // They backed out.
    });
  });
}

module.exports = {
  API_URL,
  getToken: () => token,
  setToken,
  headers,
  failure,
  clientId,
  media,
  previewPhotos,
  uploadPhoto,
  auth: {
    signUp: (b) => post('/v1/auth/signup', b),
    signIn: (email, password) => post('/v1/auth/signin', { email, password }),
    signOut: () => post('/v1/auth/signout'),
    wechat: (code, accept_terms, ref, language) => post('/v1/auth/wechat', { code, accept_terms, ref, language }),
    deleteWithWeChat: (code) => post('/v1/auth/wechat/delete', { code }),
    verify: (code) => post('/v1/auth/verify', { code }),
    resend: () => post('/v1/auth/resend'),
    forgot: (email) => post('/v1/auth/forgot', { email }),
    reset: (email, code, password) => post('/v1/auth/reset', { email, code, password }),
    changePassword: (current, next) => post('/v1/auth/change-password', { current, next }),
    sessions: async () => (await get('/v1/auth/sessions')).sessions || [],
    revokeSession: (id) => del(`/v1/auth/sessions/${id}`),
    deleteAccount: (password) => post('/v1/account/delete', { password }, 60000),
    exportData: () => get('/v1/account/export', 60000),
    me: () => get('/v1/me'),
  },
  photos: {
    list: async () => (await get('/v1/photos')).photos || [],
    setPrimary: (id) => post(`/v1/photos/${id}/primary`),
    remove: (id) => del(`/v1/photos/${id}`),
  },
  profile: { acceptBio: () => post('/v1/profile/bio-draft/accept') },
  suki: {
    messages: (after) => get('/v1/agent/messages' + (after > 0 ? `?after=${after}` : '')),
    send: (text) => post('/v1/agent/messages', { text, client_id: clientId() }),
    speechStatus: () => get('/v1/agent/speech'),
  },
  proposals: {
    list: async () => (await get('/v1/proposals')).proposals || [],
    get: async (id) => (await get(`/v1/proposals/${id}`)).proposal,
    decide: async (id, decision) => (await post(`/v1/proposals/${id}/decide`, { decision })).match_id,
  },
  matches: {
    list: async () => (await get('/v1/matches')).matches || [],
    get: async (id) => (await get(`/v1/matches/${id}`)).match,
    unmatch: (id) => post(`/v1/matches/${id}/unmatch`),
    messages: async (id, after) => (await get(`/v1/matches/${id}/messages` + (after > 0 ? `?after=${after}` : ''))).messages || [],
    send: async (id, text) => (await post(`/v1/matches/${id}/messages`, { text, client_id: clientId() })).message,
    read: (id, seq) => post(`/v1/matches/${id}/read`, { seq }),
    startDate: async (id) => (await post(`/v1/matches/${id}/date`, { constraints: [] })).date_id,
  },
  dates: {
    list: async () => (await get('/v1/dates')).dates || [],
    get: async (id) => (await get(`/v1/dates/${id}`)).date,
    vote: async (id, venue_id) => (await post(`/v1/dates/${id}/vote`, { venue_id })).date,
    time: async (id, at) => (await post(`/v1/dates/${id}/time`, { at })).date,
    confirm: async (id) => (await post(`/v1/dates/${id}/confirm`)).date,
    cancel: async (id) => (await post(`/v1/dates/${id}/cancel`, { reason: '' })).date,
    share: async (id) => (await get(`/v1/dates/${id}/share`)).text,
  },
  buddies: {
    mine: async () => (await get('/v1/buddies/mine')).requests || [],
    get: async (id) => (await get(`/v1/buddies/${id}`)).request,
    post: async (b) => (await post('/v1/buddies', b)).request,
    offer: (id) => post(`/v1/buddies/${id}/offer`),
    withdraw: (id) => del(`/v1/buddies/${id}/offer`),
    pick: async (id, user_id) => (await post(`/v1/buddies/${id}/pick`, { user_id })).match_id,
    close: (id) => post(`/v1/buddies/${id}/close`),
  },
  referral: {
    status: () => get('/v1/referral'),
  },
  credits: {
    status: () => get('/v1/credits'),
    usage: async () => (await get('/v1/credits/usage')).usage || [],
    wechatPrepay: (pack, code) => post('/v1/credits/wechat/prepay', { package: pack, code }),
  },
  safety: {
    blocked: async () => (await get('/v1/blocks')).blocked || [],
    block: (user_id) => post('/v1/blocks', { user_id }),
    unblock: (id) => del(`/v1/blocks/${id}`),
    report: (b) => post('/v1/reports', b),
  },
  settings: {
    get: async () => (await get('/v1/settings')).settings,
    patch: async (b) => (await patch('/v1/settings', b)).settings,
  },
  memories: {
    list: async () => (await get('/v1/memories')).memories || [],
    remove: (id) => del(`/v1/memories/${id}`),
    reset: () => post('/v1/memories/reset'),
  },
  notifications: {
    list: async () => (await get('/v1/notifications')).notifications || [],
    markRead: () => post('/v1/notifications/read'),
  },
};
