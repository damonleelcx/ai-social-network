const api = require('../../../lib/api');
const i18n = require('../../../lib/i18n');
const person = require('../../../lib/person');
const { day, pad, showError, confirm } = require('../../../lib/format');

const { t } = i18n;

const COPY = {
  loading: 'Getting your profile...',
  intro: 'You can also just tell me. "I\'m a nurse now" works fine.',
  about: 'About you',
  name: 'Name',
  pronouns: 'Pronouns',
  birthday: 'Birthday',
  setBirthday: 'Set your birthday',
  birthdayHint: "Wrong? Just tell Suki and she'll fix it.",
  gender: 'Gender',
  city: 'City',
  whereLive: 'Where do you live?',
  setCity: 'Set this city',
  work: 'Work',
  studied: 'Studied',
  bio: 'Bio',
  draft: "Suki's draft",
  useThis: 'Use this',
  bioPlaceholder: 'A few lines that sound like you',
  meet: 'Who you want to meet',
  like: "I'd like to meet",
  youngest: 'Youngest',
  oldest: 'Oldest',
  lookingFor: 'Looking for',
  lifestyle: 'Lifestyle',
  height: 'Height (cm)',
  smoking: 'Smoking',
  drinking: 'Drinking',
  kidsNow: 'Kids now',
  kidsFuture: 'Kids in the future',
  religion: 'Religion',
  religionHint: 'Optional, and only if you want it to matter.',
  politics: 'Politics',
  politicsHint: 'Optional, same deal.',
  languagesInterests: 'Languages and interests',
  prompts: 'Prompts',
  promptsHint: 'Up to 6. These are what people actually read.',
  question: 'Question',
  answer: 'Your answer',
  removePrompt: 'Remove this prompt',
  addPrompt: 'Add a prompt',
  dealbreakers: 'Dealbreakers',
  dealbreakersHint: "I'll never introduce you to someone on the wrong side of these.",
  save: 'Save changes',
  nothing: 'Nothing to save yet',
};
const GENDERS = { woman: 'Woman', man: 'Man', nonbinary: 'Nonbinary', other: 'Other' };
const PROMPT_IDEAS = ['My ideal Sunday', 'The way to my heart is', "I'm weirdly good at", 'A green flag I look for', 'Two truths and a lie', 'Something I want to try this year'];
const MAX_TAGS = 25;
const MAX_PROMPTS = 6;

function toForm(p) {
  return {
    display_name: p.display_name || '',
    pronouns: p.pronouns || '',
    bio: p.bio || '',
    occupation: p.occupation || '',
    education: p.education || '',
    religion: p.religion || '',
    politics: p.politics || '',
    gender: p.gender || '',
    seeking: p.seeking || [],
    age_min: p.age_min ? String(p.age_min) : '',
    age_max: p.age_max ? String(p.age_max) : '',
    goal: p.goal || '',
    height_cm: p.height_cm ? String(p.height_cm) : '',
    smoking: p.smoking || '',
    drinking: p.drinking || '',
    kids: p.kids || '',
    wants_kids: p.wants_kids || '',
    languages: p.languages || [],
    interests: p.interests || [],
    prompts: p.prompts || [],
    dealbreakers: p.dealbreakers || {},
  };
}

const same = (a, b) => JSON.stringify(a) === JSON.stringify(b);

// Only what changed. An enum can't be set back to empty, so an unset one is never sent.
function diff(before, after) {
  const patch = {};
  Object.keys(after).forEach((k) => {
    if (same(before[k], after[k])) return;
    if (k === 'age_min' || k === 'age_max' || k === 'height_cm') {
      if (after[k] !== '') patch[k] = Number(after[k]);
    } else if (k === 'prompts') {
      patch[k] = after.prompts.map((p) => ({ q: p.q.trim(), a: p.a.trim() })).filter((p) => p.q && p.a);
    } else if (k === 'dealbreakers') {
      patch[k] = {};
      Object.keys(after.dealbreakers).forEach((f) => {
        if (after.dealbreakers[f].length) patch[k][f] = after.dealbreakers[f];
      });
    } else {
      patch[k] = after[k];
    }
  });
  return patch;
}

// Edits in progress survive a reload after setting the city or the birthday.
function edits(before, current) {
  const out = {};
  Object.keys(current).forEach((k) => {
    if (!same(before[k], current[k])) out[k] = current[k];
  });
  return out;
}

const ymd = (d) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;

function marked(table, chosen) {
  return person.options(table).map((o) => Object.assign(o, { on: chosen.indexOf(o.value) >= 0 }));
}

Page({
  data: {
    t: i18n.strings(COPY),
    profile: null,
    form: null,
    dirty: false,
    error: '',
    saving: false,
    birthText: '',
    birthDraft: '',
    today: '',
    cityQuery: '',
    cityHint: '',
    cityBusy: false,
    cityError: '',
    acceptingBio: false,
    genders: [],
    goals: [],
    habits: [],
    kids: [],
    wantsKids: [],
    seeking: [],
    tags: [],
    dealbreakerPicks: [],
    dealbreakerTags: [],
    drafts: {},
    canAddPrompt: false,
  },
  onLoad() {
    const now = new Date();
    const eighteenAgo = new Date(now.getFullYear() - 18, now.getMonth(), now.getDate());
    this.setData({ today: ymd(now), birthDraft: ymd(eighteenAgo) });
    this.load();
  },
  onShow() {
    i18n.apply(this, COPY, 'Edit profile');
  },
  async load() {
    try {
      const { profile } = await api.profile.get();
      const fresh = toForm(profile);
      const kept = this.before && this.data.form ? edits(this.before, this.data.form) : {};
      this.before = fresh;
      this.setData({
        profile,
        error: '',
        birthText: profile.birthdate ? day(profile.birthdate) : '',
        cityHint: profile.city_name ? t('Now: {name}', { name: profile.city_name }) : t('Add the country if I can’t find it.'),
        genders: person.options(GENDERS),
        goals: person.options(person.GOALS),
        habits: person.options(person.HABITS),
        kids: person.options(person.KIDS),
        wantsKids: person.options(person.WANTS_KIDS),
      });
      this.show(Object.assign({}, fresh, kept));
    } catch (e) {
      this.setData({ error: e.message });
    }
  },
  // The lists that wxml can't work out by itself, because it can't call a function.
  show(form) {
    const db = form.dealbreakers;
    const picks = (key, label, table) => ({ key, label: t(label), options: marked(table, db[key] || []) });
    const tagRow = (key, label, placeholder, hint, values) => ({ key, label: t(label), placeholder: t(placeholder), hint: hint ? t(hint) : '', values });
    this.setData({
      form,
      dirty: Object.keys(diff(this.before, form)).length > 0,
      canAddPrompt: form.prompts.length < MAX_PROMPTS,
      seeking: marked(GENDERS, form.seeking),
      tags: [
        tagRow('languages', 'Languages', 'English, Korean', '', form.languages),
        tagRow('interests', 'Interests', 'Climbing, jazz, ramen', 'At least 3, so I have something to work with.', form.interests),
      ],
      dealbreakerPicks: [
        picks('smoking', "Won't date someone who smokes...", person.HABITS),
        picks('drinking', "Won't date someone who drinks...", person.HABITS),
        picks('kids', "Won't date someone who...", person.KIDS),
        picks('wants_kids', 'On future kids, not someone who...', person.WANTS_KIDS),
      ],
      dealbreakerTags: [
        tagRow('religion', 'Religions to rule out', 'Type and tap done', '', db.religion || []),
        tagRow('politics', 'Politics to rule out', 'Type and tap done', '', db.politics || []),
      ],
    });
  },
  // Typing changes one field, so only that goes back to the page. The whole form on every key would lag.
  type(path, value) {
    this.setData({ [path]: value });
    this.setData({ dirty: Object.keys(diff(this.before, this.data.form)).length > 0 });
  },
  onField(e) {
    this.type('form.' + e.currentTarget.dataset.f, e.detail.value);
  },
  onNumber(e) {
    const value = e.detail.value.replace(/\D/g, '');
    this.type('form.' + e.currentTarget.dataset.f, value);
    return value;
  },
  onPick(e) {
    const { f, v } = e.currentTarget.dataset;
    this.show(Object.assign({}, this.data.form, { [f]: v }));
  },
  onSeeking(e) {
    const v = e.currentTarget.dataset.v;
    const list = this.data.form.seeking;
    this.show(Object.assign({}, this.data.form, { seeking: list.indexOf(v) >= 0 ? list.filter((x) => x !== v) : list.concat(v) }));
  },
  setDealbreaker(key, values) {
    this.show(Object.assign({}, this.data.form, { dealbreakers: Object.assign({}, this.data.form.dealbreakers, { [key]: values }) }));
  },
  onDealbreaker(e) {
    const { key, v } = e.currentTarget.dataset;
    const list = this.data.form.dealbreakers[key] || [];
    this.setDealbreaker(key, list.indexOf(v) >= 0 ? list.filter((x) => x !== v) : list.concat(v));
  },
  onTagDraft(e) {
    this.setData({ ['drafts.' + e.currentTarget.dataset.key]: e.detail.value });
  },
  addTags(current, key) {
    const next = current.slice();
    (this.data.drafts[key] || '').split(',').map((s) => s.trim()).filter(Boolean).forEach((p) => {
      if (next.length < MAX_TAGS && !next.some((x) => x.toLowerCase() === p.toLowerCase())) next.push(p);
    });
    this.setData({ ['drafts.' + key]: '' });
    return next;
  },
  onTagAdd(e) {
    const key = e.currentTarget.dataset.key;
    if ((this.data.drafts[key] || '').trim()) this.show(Object.assign({}, this.data.form, { [key]: this.addTags(this.data.form[key], key) }));
  },
  onTagRemove(e) {
    const { key, v } = e.currentTarget.dataset;
    this.show(Object.assign({}, this.data.form, { [key]: this.data.form[key].filter((x) => x !== v) }));
  },
  onDealTagAdd(e) {
    const key = e.currentTarget.dataset.key;
    if ((this.data.drafts[key] || '').trim()) this.setDealbreaker(key, this.addTags(this.data.form.dealbreakers[key] || [], key));
  },
  onDealTagRemove(e) {
    const { key, v } = e.currentTarget.dataset;
    this.setDealbreaker(key, (this.data.form.dealbreakers[key] || []).filter((x) => x !== v));
  },
  onPrompt(e) {
    const { i, f } = e.currentTarget.dataset;
    this.type(`form.prompts[${i}].${f}`, e.detail.value);
  },
  onPromptAdd() {
    const prompts = this.data.form.prompts;
    const used = prompts.map((p) => p.q);
    const q = PROMPT_IDEAS.map((x) => t(x)).find((x) => used.indexOf(x) < 0) || '';
    this.show(Object.assign({}, this.data.form, { prompts: prompts.concat({ q, a: '' }) }));
  },
  onPromptRemove(e) {
    const i = e.currentTarget.dataset.i;
    this.show(Object.assign({}, this.data.form, { prompts: this.data.form.prompts.filter((_, j) => j !== i) }));
  },
  // Saved right away, as in the app, because people can't change it afterwards.
  async onBirthdate(e) {
    const value = e.detail.value;
    const yes = await confirm(t('Is {time} right?', { time: day(value) }), t("Please check it. SynAgent is for people 18 and over. If it's ever wrong, just tell Suki."));
    if (!yes) return;
    try {
      await api.profile.save({ birthdate: value });
    } catch (err) {
      showError(err);
    }
    this.load();
  },
  onCityInput(e) {
    this.setData({ cityQuery: e.detail.value });
  },
  async onCity() {
    this.setData({ cityBusy: true, cityError: '' });
    try {
      await api.profile.setCity(this.data.cityQuery.trim());
      this.setData({ cityQuery: '' });
      await this.load();
    } catch (e) {
      this.setData({ cityError: e.message });
    }
    this.setData({ cityBusy: false });
  },
  async onAcceptBio() {
    this.setData({ acceptingBio: true });
    try {
      await api.profile.acceptBio();
      this.setData({ 'form.bio': this.data.profile.bio_draft });
      await this.load();
    } catch (e) {
      showError(e);
    }
    this.setData({ acceptingBio: false });
  },
  async onSave() {
    this.setData({ saving: true, error: '' });
    try {
      await api.profile.save(diff(this.before, this.data.form));
      wx.navigateBack();
    } catch (e) {
      this.setData({ error: e.message });
    }
    this.setData({ saving: false });
  },
});
