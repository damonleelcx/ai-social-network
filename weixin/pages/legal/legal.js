// The terms, privacy policy and the other legal pages, shown inside the mini program. They are
// the same pages the website serves; a web-view would need a filed business domain, so the HTML
// is fetched and drawn with rich-text instead. Anyone can read them, signed in or not.
const i18n = require('../../lib/i18n');
const { API_URL } = require('../../config');

const { t } = i18n;

const COPY = {
  loading: 'Loading...',
  failed: "This page didn't load. Check your connection and try again.",
  retry: 'Try again',
  copy: 'Copy the link',
};

const PAGES = ['terms', 'privacy', 'contact', 'delete-account'];

// rich-text takes no stylesheet, so each tag gets its look inline.
const LOOK = {
  h1: 'font-size:22px;font-weight:600;line-height:1.3;margin:0 0 8px;',
  h2: 'font-size:18px;font-weight:600;margin:24px 0 8px;',
  h3: 'font-size:16px;font-weight:600;margin:16px 0 4px;',
  p: 'margin:8px 0;line-height:1.7;',
  ul: 'padding-left:20px;margin:8px 0;',
  ol: 'padding-left:20px;margin:8px 0;',
  li: 'margin:4px 0;line-height:1.7;',
  table: 'border-collapse:collapse;width:100%;font-size:14px;margin:8px 0;',
  th: 'text-align:left;vertical-align:top;border-bottom:1px solid #352a55;padding:6px 8px 6px 0;',
  td: 'text-align:left;vertical-align:top;border-bottom:1px solid #352a55;padding:6px 8px 6px 0;',
  a: 'color:#bfa2ff;',
};

// The page body, without its scripts, styles and site footer, with the look written in.
function shape(html) {
  const title = (html.match(/<title>([\s\S]*?)<\/title>/i) || [])[1] || '';
  let body = (html.match(/<body[^>]*>([\s\S]*)$/i) || [])[1] || html;
  body = body
    .replace(/<\/body>[\s\S]*$/i, '')
    .replace(/<script[\s\S]*?<\/script>/gi, '')
    .replace(/<style[\s\S]*?<\/style>/gi, '')
    .replace(/<footer[\s\S]*?<\/footer>/gi, '')
    .replace(/<p class="meta">/g, '<p style="color:#b3a8cc;font-size:13px;margin:0 0 16px;">')
    .replace(/<(h1|h2|h3|p|ul|ol|li|table|th|td|a)(\s[^>]*)?>/gi, (m, tag, rest) => {
      const look = LOOK[tag.toLowerCase()];
      const attrs = (rest || '').replace(/\sstyle="[^"]*"/i, '');
      return /style=/.test(m) && tag.toLowerCase() === 'p' ? m : `<${tag}${attrs} style="${look}">`;
    });
  return { title: title.trim(), body };
}

Page({
  data: { t: i18n.strings(COPY), page: 'privacy', html: '', state: 'loading' },
  onLoad(q) {
    this.setData({ page: PAGES.indexOf(q.page) >= 0 ? q.page : 'privacy' });
    this.load();
  },
  onShow() {
    i18n.apply(this, COPY);
  },
  url() {
    return `${API_URL}/legal/${this.data.page}?lang=${i18n.current()}`;
  },
  load() {
    this.setData({ state: 'loading' });
    wx.request({
      url: this.url(),
      method: 'GET',
      dataType: 'text',
      responseType: 'text',
      success: (r) => {
        if (r.statusCode !== 200 || typeof r.data !== 'string') {
          this.setData({ state: 'failed' });
          return;
        }
        const { title, body } = shape(r.data);
        if (title) wx.setNavigationBarTitle({ title });
        this.setData({ html: body, state: 'ready' });
      },
      fail: () => this.setData({ state: 'failed' }),
    });
  },
  onRetry() {
    this.load();
  },
  onCopy() {
    wx.setClipboardData({ data: this.url() });
  },
});
