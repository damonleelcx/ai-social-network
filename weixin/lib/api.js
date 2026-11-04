// The server's API, over wx.request. The token lives in this mini program's own storage.
const { API_URL } = require('../config');

let token = wx.getStorageSync('token') || '';
let leaving = false;

const ZH = {
  'invalid email or password': '邮箱或密码不正确',
  'an account with this email already exists': '这个邮箱已经注册过了',
  'password must be at least 10 characters': '密码至少需要 10 位',
  'that email address does not look valid': '邮箱格式不正确',
  'that code is wrong or has expired': '验证码不正确或已过期',
  'too many attempts; try again in 15 minutes': '尝试次数太多，请 15 分钟后再试',
  'too many codes requested; try again in an hour': '验证码请求太多，请一小时后再试',
  'please accept the terms to continue': '请先同意用户协议',
  "this email can't be used on SynAgent": '这个邮箱不能使用 SynAgent',
  'Slow down a little and try again in a moment.': '操作太频繁，请稍后再试。',
};

function setToken(t) {
  token = t || '';
  if (token) wx.setStorageSync('token', token);
  else wx.removeStorageSync('token');
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
  const err = new Error(ZH[message] || message || '出了点问题，请再试一次。');
  err.status = status;
  return err;
}

function headers(extra) {
  return Object.assign({ Accept: 'application/json' }, token ? { Authorization: 'Bearer ' + token } : {}, extra);
}

function request(path, method, body) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: API_URL + path,
      method: method || 'GET',
      data: body,
      timeout: 20000,
      header: headers(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
      success(res) {
        if (res.statusCode >= 200 && res.statusCode < 300) resolve(res.data);
        else reject(failure(res.statusCode, res.data && res.data.error));
      },
      fail() {
        reject(new Error('连接不上服务器，请检查网络后再试。'));
      },
    });
  });
}
const get = (p) => request(p);
const post = (p, b) => request(p, 'POST', b === undefined ? {} : b);
const del = (p) => request(p, 'DELETE');

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
              fail: () => reject(new Error('上传失败，请再试一次。')),
            });
          },
          fail: () => reject(new Error('读取不了这张照片。')),
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
  clientId,
  media,
  uploadPhoto,
  auth: {
    signUp: (b) => post('/v1/auth/signup', b),
    signIn: (email, password) => post('/v1/auth/signin', { email, password }),
    signOut: () => post('/v1/auth/signout'),
    verify: (code) => post('/v1/auth/verify', { code }),
    resend: () => post('/v1/auth/resend'),
    forgot: (email) => post('/v1/auth/forgot', { email }),
    reset: (email, code, password) => post('/v1/auth/reset', { email, code, password }),
    deleteAccount: (password) => post('/v1/account/delete', { password }),
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
  credits: {
    status: () => get('/v1/credits'),
    usage: async () => (await get('/v1/credits/usage')).usage || [],
    wechatPrepay: (pack, code) => post('/v1/credits/wechat/prepay', { package: pack, code }),
  },
  safety: {
    block: (user_id) => post('/v1/blocks', { user_id }),
    report: (b) => post('/v1/reports', b),
  },
  memories: { reset: () => post('/v1/memories/reset') },
};
