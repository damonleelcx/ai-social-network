// Suki's voice, as in the app: one clip at a time, kept on disk until sign-out.
const api = require('./api');
const i18n = require('./i18n');
const { showError } = require('./format');

const MAX_CACHED = 60;
const REQUEST_MS = 60000;
const dir = () => `${wx.env.USER_DATA_PATH}/suki-voice`;

// enabled: does the server have a voice at all? Everything voice-related hides when it doesn't.
// clip: a message's seq, or 'sample' for the fixed greeting.
// tapped: the person asked for this one, so autoplay must not talk over it.
let state = { enabled: false, clip: null, phase: null, tapped: false };
const listeners = new Set();
let generation = 0;
let audio = null;
let task = null;

function update(next) {
  state = Object.assign({}, state, next);
  listeners.forEach((l) => l(state));
}

const exists = (path) => new Promise((resolve) => {
  wx.getFileSystemManager().access({ path, success: () => resolve(true), fail: () => resolve(false) });
});

function prepareDir() {
  const fs = wx.getFileSystemManager();
  try {
    fs.mkdirSync(dir(), true);
  } catch (_) {
    // It's already there.
  }
  try {
    const names = fs.readdirSync(dir());
    if (names.length >= MAX_CACHED) names.forEach((n) => fs.unlinkSync(`${dir()}/${n}`));
  } catch (_) {
    // A leftover clip is harmless.
  }
}

// An error comes back as bytes too, because the request asked for bytes. The server's
// messages are JSON and ASCII.
function errorIn(buffer) {
  try {
    const text = String.fromCharCode.apply(null, new Uint8Array(buffer));
    return JSON.parse(text).error || '';
  } catch (_) {
    return '';
  }
}

function fetchClip(key, path) {
  return new Promise((resolve, reject) => {
    task = wx.request({
      url: `${api.API_URL}/v1/agent/speech`,
      method: 'POST',
      data: key === 'sample' ? { sample: true } : { seq: key },
      responseType: 'arraybuffer',
      timeout: REQUEST_MS,
      header: api.headers({ 'Content-Type': 'application/json' }),
      success(res) {
        if (res.statusCode < 200 || res.statusCode >= 300) {
          // The server says voice is off, so stop offering it.
          if (res.statusCode === 503) update({ enabled: false });
          reject(api.failure(res.statusCode, errorIn(res.data)));
          return;
        }
        prepareDir();
        wx.getFileSystemManager().writeFile({
          filePath: path,
          data: res.data,
          success: () => resolve(path),
          fail: () => reject(new Error(i18n.t("Suki's voice is unavailable right now"))),
        });
      },
      fail: () => reject(new Error(i18n.t("I can't reach the server right now. Check your connection and try again."))),
    });
  });
}

async function clipFile(key) {
  // The greeting is said in the person's language, so each language gets its own file.
  const path = `${dir()}/${key === 'sample' ? 'sample-' + i18n.current() : key}.mp3`;
  if (await exists(path)) return path;
  return fetchClip(key, path);
}

const voice = {
  async refreshEnabled() {
    try {
      const { enabled } = await api.suki.speechStatus();
      update({ enabled: !!enabled });
    } catch (_) {
      update({ enabled: false });
    }
  },
  stop() {
    generation += 1;
    if (task) task.abort();
    task = null;
    const a = audio;
    audio = null;
    if (a) {
      a.stop();
      a.destroy();
    }
    if (state.clip !== null) update({ clip: null, phase: null, tapped: false });
  },
  // Replaces whatever was playing. `tapped` means the person asked, so it sounds even with
  // the iPhone's silent switch on. Autoplay respects the switch. Rejects with the API's
  // error, unless a newer call or stop() took over.
  async play(key, opts) {
    const tapped = !!(opts && opts.tapped);
    voice.stop();
    generation += 1;
    const mine = generation;
    update({ clip: key, phase: 'loading', tapped });
    try {
      const path = await clipFile(key);
      task = null;
      if (mine !== generation) return;
      wx.setInnerAudioOption({ obeyMuteSwitch: !tapped, fail: () => {} });
      const a = wx.createInnerAudioContext();
      audio = a;
      a.onEnded(() => {
        if (mine === generation) voice.stop();
      });
      a.onError(() => {
        if (mine !== generation) return;
        voice.stop();
        if (tapped) showError(new Error(i18n.t("Suki's voice is unavailable right now")));
      });
      a.src = path;
      a.play();
      update({ phase: 'playing' });
    } catch (e) {
      if (mine !== generation) return;
      voice.stop();
      throw e;
    }
  },
  // Clips are the person's own conversation, so they don't outlive the session.
  clear() {
    voice.stop();
    wx.getFileSystemManager().rmdir({ dirPath: dir(), recursive: true, fail: () => {} });
    update({ enabled: false });
  },
  current: () => state,
  on(listener) {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },
};

module.exports = voice;
