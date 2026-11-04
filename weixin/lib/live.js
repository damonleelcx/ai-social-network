// One socket and a small bus. Pages subscribe by topic and refetch. If WeChat drops the
// socket, the pages' own polling still gets them there.
const { API_URL } = require('../config');

const listeners = new Set();
let task = null;
let wanted = false;
let attempt = 0;
let timer = null;

function emit(topic, data) {
  listeners.forEach((l) => l(topic, data));
}

function open() {
  const token = wx.getStorageSync('token');
  if (!wanted || !token || task) return;
  const socket = wx.connectSocket({
    url: API_URL.replace(/^http/, 'ws') + '/v1/ws',
    header: { Authorization: 'Bearer ' + token },
  });
  task = socket;
  socket.onOpen(() => {
    if (attempt > 0) emit('reconnect');
    attempt = 0;
  });
  socket.onMessage((e) => {
    try {
      const { topic, data } = JSON.parse(e.data);
      if (typeof topic === 'string') emit(topic, data);
    } catch (_) {
      // Not a frame we understand. The next refetch covers it.
    }
  });
  socket.onClose(() => {
    if (task === socket) task = null;
    if (!wanted) return;
    attempt += 1;
    clearTimeout(timer);
    timer = setTimeout(open, Math.min(30000, 1000 * 2 ** attempt));
  });
}

module.exports = {
  connect() {
    wanted = true;
    if (!task) open();
  },
  disconnect() {
    wanted = false;
    clearTimeout(timer);
    if (task) task.close({});
    task = null;
  },
  on(listener) {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },
};
