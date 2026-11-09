const api = require('../../lib/api');
const live = require('../../lib/live');
const i18n = require('../../lib/i18n');
const settings = require('../../lib/settings');
const voice = require('../../lib/voice');
const { showError } = require('../../lib/format');
const share = require('../../lib/share');

const POLL_MS = 3000;
const POLL_FOR_MS = 180000;

const COPY = {
  aiGenerated: 'AI-generated',
  winggirl: 'Your winggirl. Openly an AI.',
  thinkingHeader: 'Thinking about what you said...',
  settings: "Suki's settings",
  hi: "Hi, I'm Suki!",
  hiBody: "I'm an AI matchmaker, and I'm on your side. Say hi and we'll get to know each other. No forms, I promise.",
  loading: 'Finding our conversation...',
  placeholder: 'Tell Suki anything',
  send: 'Send',
  hear: 'Hear Suki say this',
  stop: 'Stop',
  cancelVoice: 'Getting Suki’s voice. Tap to cancel',
  momentsTitle: "Hi, I'm Suki!",
  momentsBody: 'I help you find company for a meal, a film, a walk or a trip, and introduce you to people worth meeting. Tap the button below to open SynAgent and say hi.',
  guestBody: 'Have a look around. When you want to talk to me, sign in and we can start.',
  whatIDo: 'What I can do for you',
  doCompany: 'Find company for a meal, a film, a walk or a trip',
  doIntro: 'Introduce you to people worth meeting, one at a time',
  doPlan: 'Plan the meetup: a place you both like and a time that works',
  doSafe: 'Keep it safe: report or block anyone, any time',
  signInToChat: 'Sign in to chat with Suki',
};

function shape(m) {
  const meta = m.meta || {};
  return { seq: m.seq, role: m.role, body: m.body, mood: meta.mood || '', cards: meta.cards || [], replies: meta.replies || [] };
}

// What her face in the header shows: what she last felt, or that she's thinking.
function headerMood(messages, thinking) {
  if (thinking) return 'thinking';
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].role === 'assistant' && messages[i].mood) return messages[i].mood;
  }
  return 'happy';
}

function newestSpeakable(messages) {
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].role === 'assistant' && messages[i].body) return messages[i].seq;
  }
  return 0;
}

Page({
  data: { t: i18n.strings(COPY), guest: false, moments: false, messages: [], thinking: false, draft: '', replies: [], bottom: '', loaded: false, speaker: false, clip: null, phase: null, moodAvatar: true, headerMood: 'happy' },
  onLoad() {
    // Opened from Moments: no sign-in and no navigation here, just a hello. WeChat shows its
    // own button to open the full mini program, which keeps the link's invite code.
    if (share.fromMoments()) {
      this.setData({ moments: true });
      return;
    }
    wx.showShareMenu({ menus: ['shareAppMessage', 'shareTimeline'], fail: () => {} });
    // Not signed in: they can look around, and sign in when they choose to.
    if (!api.getToken()) {
      this.setData({ guest: true, loaded: true });
      return;
    }
    this.off = live.on((topic) => {
      if (topic === 'agent' || topic === 'reconnect') this.load();
      else if (topic === 'typing:agent') this.watch();
    });
    this.offVoice = voice.on((s) => {
      this.setData({ clip: s.clip, phase: s.phase });
      this.showSpeaker();
    });
  },
  onUnload() {
    if (this.off) this.off();
    if (this.offVoice) this.offVoice();
    this.stopPolling();
  },
  onShow() {
    i18n.apply(this, COPY);
    if (this.data.moments) return;
    i18n.tabs();
    this.focused = true;
    // Only what arrives from here on is new. Nothing she said while they were away gets read out.
    this.spokenUpTo = null;
    if (api.getToken()) {
      this.load();
      share.prepare();
      voice.refreshEnabled();
      settings.load().then(() => this.showSpeaker(), () => {});
    }
  },
  onShareAppMessage() {
    return share.message();
  },
  onShareTimeline() {
    return share.timeline();
  },
  onHide() {
    this.focused = false;
    voice.stop();
    this.stopPolling();
  },
  showSpeaker() {
    const s = settings.current();
    const speaker = voice.current().enabled && (s ? s.suki_voice : true);
    // Autoplay needs the speakers too, or there'd be nothing on screen to stop it with.
    this.autoplay = speaker && !!(s && s.suki_voice_auto);
    if (speaker !== this.data.speaker) this.setData({ speaker });
    const moodAvatar = !s || s.mood_avatar !== false;
    if (moodAvatar !== this.data.moodAvatar) this.setData({ moodAvatar });
  },
  readAloud(messages) {
    const newest = newestSpeakable(messages);
    if (this.spokenUpTo === null || this.spokenUpTo === undefined) {
      this.spokenUpTo = newest;
      return;
    }
    if (newest <= this.spokenUpTo) return;
    this.spokenUpTo = newest;
    // Never talk over a clip the person asked for.
    if (!this.autoplay || !this.focused || voice.current().tapped) return;
    // Autoplay fails quietly: running out of allowance shouldn't interrupt the chat.
    voice.play(newest, { tapped: false }).catch(() => {});
  },
  onSpeak(e) {
    const seq = e.currentTarget.dataset.seq;
    if (this.data.clip === seq) voice.stop();
    else voice.play(seq, { tapped: true }).catch(showError);
  },
  async load() {
    if (!api.getToken()) return;
    const last = this.data.messages[this.data.messages.length - 1];
    try {
      const r = await api.suki.messages(last ? last.seq : 0);
      const seen = new Set(this.data.messages.map((m) => m.seq));
      const fresh = (r.messages || []).map(shape).filter((m) => !seen.has(m.seq));
      const messages = this.data.messages.concat(fresh);
      const tail = messages[messages.length - 1];
      this.setData({
        messages,
        thinking: r.thinking,
        headerMood: headerMood(messages, r.thinking),
        loaded: true,
        replies: tail && tail.role !== 'user' ? tail.replies : [],
        bottom: fresh.length ? 'end' : this.data.bottom,
      });
      this.readAloud(messages);
      if (r.thinking) this.watch();
      else this.stopPolling();
    } catch (e) {
      if (e.status !== 401 && e.status !== 403) showError(e);
    }
  },
  // The socket says when she's done. This is for when it can't.
  watch() {
    if (this.poll) return;
    const until = Date.now() + POLL_FOR_MS;
    this.poll = setInterval(() => {
      if (Date.now() > until) this.stopPolling();
      else this.load();
    }, POLL_MS);
  },
  stopPolling() {
    clearInterval(this.poll);
    this.poll = null;
  },
  onSignIn() {
    wx.navigateTo({ url: '/pages/login/login' });
  },
  onLegal(e) {
    wx.navigateTo({ url: '/pages/legal/legal?page=' + e.currentTarget.dataset.page });
  },
  onSettings() {
    if (this.data.guest) return this.onSignIn();
    wx.navigateTo({ url: '/pages/settings/suki/suki' });
  },
  onInput(e) {
    this.setData({ draft: e.detail.value });
  },
  onReply(e) {
    this.send(e.currentTarget.dataset.text);
  },
  // A card asked Suki something on this person's behalf, such as more about someone introduced.
  onCardAsk(e) {
    if (e.detail && e.detail.text) this.send(e.detail.text);
  },
  onSend() {
    if (this.data.guest) return this.onSignIn();
    const text = this.data.draft.trim();
    if (!text) return;
    this.setData({ draft: '' });
    this.send(text);
  },
  async send(text) {
    this.setData({ replies: [] });
    try {
      await api.suki.send(text);
      this.setData({ thinking: true, headerMood: 'thinking' });
      this.watch();
      await this.load();
      this.watch();
    } catch (e) {
      this.setData({ draft: text });
      showError(e);
    }
  },
});
