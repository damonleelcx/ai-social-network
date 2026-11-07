// Copy is written in English and looked up by its English text, the same way as the app.
// The default is WeChat's own language (Chinese if we don't have it). Once someone is
// signed in, their language setting decides.
//
// wxml can't call a function, so each page lists its words in COPY, and apply() turns that
// into data.t for the page's language. Words with a value in them are built in the page's
// load() with t(), and load() runs again when the language changes.
const catalog = require('./catalog');

const COLUMN = { zh: 0, ko: 1, ja: 2, es: 3 };
const LANGUAGES = [
  { value: 'zh', label: '中文' },
  { value: 'en', label: 'English' },
  { value: 'ko', label: '한국어' },
  { value: 'ja', label: '日本語' },
  { value: 'es', label: 'Español' },
];
const TABS = [null, 'Matches', 'Dates', 'Me']; // The first is Suki, in every language.
// wx.showModal refuses buttons longer than four Chinese characters, so these are kept short.
const MODAL = {
  zh: ['取消', '确定'],
  en: ['Cancel', 'OK'],
  ko: ['취소', '확인'],
  ja: ['やめる', 'OK'],
  es: ['No', 'Sí'],
};
const KEY = 'language';

const isLanguage = (l) => typeof l === 'string' && (l === 'en' || COLUMN[l] !== undefined);

// WeChat's own language (zh_CN, en, ko, ja ...) when nothing is chosen yet; Chinese if unknown.
function deviceLanguage() {
  let tag = '';
  try {
    tag = (wx.getAppBaseInfo ? wx.getAppBaseInfo() : wx.getSystemInfoSync()).language || '';
  } catch (e) {
    return 'zh';
  }
  const base = String(tag).toLowerCase().split(/[-_]/)[0];
  return isLanguage(base) ? base : 'zh';
}

let current = wx.getStorageSync(KEY);
if (!isLanguage(current)) current = deviceLanguage();
let tabsIn = '';
const parts = new Set();

function t(english, vars) {
  const row = catalog[english];
  let text = Array.isArray(row) && COLUMN[current] !== undefined ? row[COLUMN[current]] : english;
  Object.keys(vars || {}).forEach((k) => {
    text = text.split('{' + k + '}').join(String(vars[k]));
  });
  return text;
}

function strings(copy) {
  const out = {};
  Object.keys(copy).forEach((k) => {
    out[k] = t(copy[k]);
  });
  return out;
}

function render(view) {
  view.i18nIn = current;
  view.setData({ t: strings(view.i18nCopy) });
  if (view.i18nTitle) wx.setNavigationBarTitle({ title: t(view.i18nTitle) });
}

// For a page's onShow, so a page that was hidden when the language changed catches up.
function apply(view, copy, title) {
  view.i18nCopy = copy;
  view.i18nTitle = title;
  if (view.i18nIn !== current) render(view);
}

// WeChat only lets a tab page do this, so the tab pages call it from onShow.
function tabs() {
  if (tabsIn === current) return;
  const lang = current;
  TABS.forEach((text, index) => {
    if (!text) return;
    wx.setTabBarItem({
      index,
      text: t(text),
      success: () => {
        tabsIn = lang;
      },
      fail: () => {},
    });
  });
}

// A component stays on screen when the language changes, so it's told directly.
function attach(component, copy) {
  parts.add(component);
  apply(component, copy);
}

function detach(component) {
  parts.delete(component);
}

function setLanguage(language) {
  if (!isLanguage(language) || language === current) return false;
  current = language;
  wx.setStorageSync(KEY, language);
  parts.forEach(render);
  getCurrentPages().forEach((page) => {
    if (page.i18nCopy) render(page);
    if (typeof page.load === 'function') page.load();
  });
  return true;
}

function modalButtons() {
  const [cancelText, confirmText] = MODAL[current];
  return { cancelText, confirmText };
}

module.exports = { t, strings, apply, tabs, attach, detach, setLanguage, isLanguage, modalButtons, LANGUAGES, current: () => current };
