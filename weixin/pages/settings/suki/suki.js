const i18n = require('../../../lib/i18n');
const settings = require('../../../lib/settings');
const voice = require('../../../lib/voice');
const { showError } = require('../../../lib/format');

const { t } = i18n;

const COPY = {
  loading: 'Getting your settings...',
  intro: "These change how I come across, not who I am. I'll always be on your side, and always honest that I'm an AI.",
  howSheTalks: 'How she talks',
  tone: 'Tone',
  emoji: 'Emoji',
  checksIn: 'How often she checks in',
  calls: 'What she calls you',
  callsPlaceholder: 'Your name, unless you say otherwise',
  herVoice: 'Her voice',
  speakers: 'Speaker buttons',
  speakersHint: 'Tap the speaker on a message to hear it.',
  aloud: 'Read new messages aloud',
  aloudHint: "Only while you're on the Suki tab, and not when your phone is on silent.",
  hear: 'Hear Suki',
  stop: 'Stop',
  yourChats: 'Your chats',
  coaching: 'Coaching',
  coachingHint: 'Private hints in your chats. Only you see them.',
  coachRead: 'Let Suki read my chats',
  coachReadHint: 'She reads your messages to give better hints. Off means general tips only.',
  prompts: 'Chat prompts',
  promptsHint: 'She can post a question for you both when a chat goes quiet.',
  memory: 'Memory',
  longTerm: 'Long-term memory',
  longTermHint: 'She keeps notes between chats.',
  debriefs: 'Date debriefs',
  debriefsHint: 'She asks how it went after a date.',
  remembers: 'What Suki remembers',
  remembersHint: 'See, delete or start over',
  safety: 'Safety',
  blocked: 'Blocked people',
};
const TONES = [['playful', 'Playful'], ['warm', 'Warm'], ['direct', 'Direct'], ['calm', 'Calm']];
const EMOJI = [['none', 'None'], ['some', 'Some'], ['lots', 'Lots']];
const PROACTIVE = [['low', 'Only when I ask'], ['normal', 'Now and then'], ['high', 'Keep me posted']];

const options = (list) => list.map((o) => ({ value: o[0], label: t(o[1]) }));

Page({
  data: { t: i18n.strings(COPY), s: null, tones: [], emoji: [], proactive: [], voiceOn: false, clip: null, phase: null },
  onLoad() {
    this.off = voice.on((v) => this.setData({ voiceOn: v.enabled, clip: v.clip, phase: v.phase }));
  },
  onUnload() {
    if (this.off) this.off();
    voice.stop();
  },
  onHide() {
    voice.stop();
  },
  onShow() {
    i18n.apply(this, COPY, "Suki's settings");
    this.load();
    voice.refreshEnabled();
  },
  async load() {
    this.setData({ tones: options(TONES), emoji: options(EMOJI), proactive: options(PROACTIVE) });
    try {
      this.setData({ s: await settings.load() });
    } catch (e) {
      if (e.status !== 401) showError(e);
    }
  },
  // Shows the change right away, and goes back to what's saved if it doesn't save.
  async save(change) {
    this.setData({ s: Object.assign({}, this.data.s, change) });
    try {
      this.setData({ s: await settings.patch(change) });
    } catch (e) {
      this.setData({ s: settings.current() });
      showError(e);
    }
  },
  onToggle(e) {
    this.save({ [e.currentTarget.dataset.f]: e.detail.value });
  },
  onPick(e) {
    const { f, v } = e.currentTarget.dataset;
    if (this.data.s[f] !== v) this.save({ [f]: v });
  },
  onNickname(e) {
    const value = e.detail.value.trim();
    if (value !== this.data.s.suki_nickname) this.save({ suki_nickname: value });
  },
  onHear() {
    if (this.data.clip === 'sample') voice.stop();
    else voice.play('sample', { tapped: true }).catch(showError);
  },
  onOpen(e) {
    wx.navigateTo({ url: e.currentTarget.dataset.url });
  },
});
