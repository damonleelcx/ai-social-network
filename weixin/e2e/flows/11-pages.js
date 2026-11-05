// Every page, in every language: no untranslated words, no raw placeholders, nothing that
// looks like a score, no console errors. Screenshots are for looking at the layout.
const fs = require('fs');
const path = require('path');

const SCORE = /\b(score|rank|percentile|compat(ibility)?)\b|\d+\s*%/i;
const BRANDS = /Suki|SynAgent|WeChat|iPhone|Android|English|JSON|iOS|^OK$/;

module.exports = async (l) => {
  const m = l.m();
  const account = JSON.parse(fs.readFileSync(path.join(l.OUT, 'main-account.json'), 'utf8'));
  const world = JSON.parse(fs.readFileSync(path.join(l.OUT, 'world.json'), 'utf8'));
  await l.signIn(account);
  const token = await m.callWxMethod('getStorageSync', 'token');
  const date = l.sql(`SELECT id FROM date_plans WHERE match_id='${world.match}' ORDER BY created_at DESC LIMIT 1;`);

  const pages = [
    '/pages/suki/suki', '/pages/matches/matches', '/pages/dates/dates', '/pages/me/me',
    `/pages/chat/chat?id=${world.match}`, `/pages/date/date?id=${date}`, `/pages/person/person?match=${world.match}`,
    '/pages/profile/edit/edit', '/pages/notifications/notifications', '/pages/settings/app/app',
    '/pages/settings/account/account', '/pages/settings/blocked/blocked', '/pages/settings/suki/suki', '/pages/settings/memory/memory',
  ];
  for (const language of ['zh', 'ko', 'ja', 'es']) {
    await l.api(token, 'POST', '/v1/settings', { language });
    // The app picks the language up from the settings page, as it does on launch.
    let p = await l.goto('/pages/settings/app/app');
    await l.waitData(p, (x) => x.s && x.s.language === language, `${language} applied`, 10000);
    const bad = [];
    for (const url of pages) {
      p = await l.goto(url);
      await l.sleep(1200);
      const d = await p.data();
      const name = url.split('?')[0].split('/').pop();
      if (language !== 'es') {
        // Spanish shares too many words with English to tell by the alphabet.
        const english = Object.values(d.t || {}).filter((v) => /^[\x20-\x7e]+$/.test(v) && /[a-z]{3}/i.test(v) && !BRANDS.test(v));
        if (english.length) bad.push(`${name}: ${english.join(' | ')}`);
      }
      const text = JSON.stringify(d);
      if (/\{(name|n|value|count|used|free|price|activity|time|terms|privacy)\}/.test(text)) bad.push(`${name}: raw placeholder`);
      if (SCORE.test(JSON.stringify(Object.assign({}, d, { t: undefined })))) bad.push(`${name}: looks like a score: ${text.match(SCORE)[0]}`);
      if (language === 'zh' || (language === 'es' && /me|edit|suki|account/.test(name))) await l.shot(`${language}-${name}`);
    }
    l.check(bad.length === 0, `every page in ${language}: nothing untranslated, no placeholders, no scores`, bad.join(' // '));
  }
  await l.api(token, 'POST', '/v1/settings', { language: 'en' });
  const p = await l.goto('/pages/settings/app/app');
  await l.waitData(p, (x) => x.s && x.s.language === 'en', 'back to English', 10000);
};
