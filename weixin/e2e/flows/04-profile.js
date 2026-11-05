// Setting up a profile by talking to Suki (real model), then the edit page, then a photo.
// wx.chooseMedia can't be driven, so it's mocked to hand back a real file in the project.
const fs = require('fs');
const path = require('path');

async function editProfile(l, token) {
  let p = await l.goto('/pages/me/me');
  await l.waitData(p, (d) => d.user && d.user.id, 'Me loaded');
  await l.shot('me');
  await l.tapText(p, '.nav-row', 'Edit profile');
  p = await l.waitPage('pages/profile/edit/edit');
  let d = await l.waitData(p, (x) => x.form, 'edit form');
  await l.shot('edit-top');
  l.check(d.dirty === false, 'the form opens with nothing to save');

  await l.fill(p, 'input[data-f=occupation]', 'Architect');
  await l.fill(p, 'input[data-f=education]', 'Sorbonne');
  await l.fill(p, 'textarea[data-f=bio]', 'Jazz on weekends, trails when it is sunny.');
  await l.tapText(p, '.chip[data-f=drinking]', 'Sometimes');
  await l.tapText(p, '.chip[data-f=smoking]', 'No');
  await l.fill(p, 'input[data-f=height_cm]', '168');
  d = await p.data();
  l.check(d.dirty === true, 'typing turns on Save');

  const tag = await p.$('input[data-key=interests]');
  await tag.input('climbing, Jazz');
  await tag.trigger('confirm', { value: 'climbing, Jazz' });
  await l.sleep(300);
  d = await p.data();
  l.check(d.form.interests.some((x) => x === 'climbing'), 'a new interest is added', JSON.stringify(d.form.interests));
  l.check(d.form.interests.filter((x) => x.toLowerCase() === 'jazz').length <= 1, 'a repeated interest is not added twice', JSON.stringify(d.form.interests));
  const languages = await p.$('input[data-key=languages]');
  await languages.input('English, French');
  await languages.trigger('confirm', { value: 'English, French' });
  await l.sleep(300);

  await l.tapText(p, 'button', 'Add a prompt');
  await l.fill(p, 'textarea[data-f=a]', 'A record shop and a long walk.');
  await l.tapText(p, '.chip[data-key=smoking]', 'Yes');
  await l.sleep(200);
  await l.shot('edit-filled');

  await l.tapText(p, 'button', 'Save changes');
  await l.waitPage('pages/me/me');
  l.check(true, 'saving goes back to Me');
  const posted = await l.api(token, 'GET', '/v1/profile');
  const P = posted.data.profile;
  l.check(P.occupation === 'Architect' && P.education === 'Sorbonne', 'text fields saved', JSON.stringify([P.occupation, P.education]));
  l.check(P.height_cm === 168, 'height saved as a number', P.height_cm);
  l.check(P.drinking === 'sometimes' && P.smoking === 'no', 'choices saved', JSON.stringify([P.drinking, P.smoking]));
  l.check((P.prompts || []).length === 1, 'prompt saved', JSON.stringify(P.prompts));
  l.check(JSON.stringify(P.dealbreakers || {}).includes('yes'), 'dealbreaker saved', JSON.stringify(P.dealbreakers));

}

module.exports = async (l) => {
  const m = l.m();
  const account = JSON.parse(fs.readFileSync(path.join(l.OUT, 'main-account.json'), 'utf8'));
  await l.signIn(account);
  const token = await m.callWxMethod('getStorageSync', 'token');

  let p = await l.goto('/pages/suki/suki');
  await l.waitData(p, (d) => d.loaded, 'Suki loaded');
  let r = await l.api(token, 'GET', '/v1/profile');
  if (!r.data.profile.display_name) {
    await l.sukiSay(p, "Call me Ana. I'm a woman born on 1995-04-12, I live in Paris, and I want something long-term with a man aged 28 to 42. I love jazz, hiking and ramen.");
    await l.shot('suki-after-intro');
    const said = (await p.data()).messages;
    l.check(said[said.length - 1].role === 'assistant' && !!said[said.length - 1].body, 'Suki answers in the conversation');
    l.check(!/\{\w+\}/.test(said[said.length - 1].body), 'no raw placeholder in her reply');
    await l.sleep(1500);
    r = await l.api(token, 'GET', '/v1/profile');
  }
  l.note(`After talking: ${JSON.stringify({ name: r.data.profile.display_name, gender: r.data.profile.gender, birth: r.data.profile.birthdate, city: r.data.profile.city_name, goal: r.data.profile.goal, interests: r.data.profile.interests, seeking: r.data.profile.seeking, missing: r.data.missing })}`);
  l.check(r.data.profile.display_name === 'Ana', 'Suki saved the name from the conversation', r.data.profile.display_name);
  l.check((r.data.profile.interests || []).length > 0, 'Suki saved interests from the conversation', JSON.stringify(r.data.profile.interests));

  // Profile edit, through the page.
  if (!r.data.profile.occupation) await editProfile(l, token);


  // City, through its own button. The map lookup is seeded, because the local server may not
  // reach OpenStreetMap.
  l.seedCity();
  p = await l.goto('/pages/profile/edit/edit');
  let d = await l.waitData(p, (x) => x.form, 'edit form again');
  l.check(d.form.occupation === 'Architect', 'saved values come back on reload');
  const inputs = await p.$$('input');
  let city = null;
  for (const i of inputs) if (/Where do you live|Paris/.test((await i.attribute('placeholder')) || '')) city = i;
  await city.input('Paris, France');
  await l.tapText(p, 'button', 'Set this city', 4000);
  d = await l.waitData(p, (x) => !x.cityBusy && (x.cityError || x.profile.city_name), 'city result', 30000);
  l.check(!d.cityError && /Paris/.test(d.profile.city_name), 'the city resolves through the local server', d.cityError || d.profile.city_name);
  await l.shot('edit-city');
  l.note(`City: ${d.profile.city_name}, error: ${d.cityError}`);

  // Photo. Needs a file DevTools can read, so run.js's README has you put one in the copy.
  p = await l.goto('/pages/me/me');
  await l.waitData(p, (x) => x.user && x.user.id, 'Me loaded');
  await m.mockWxMethod('chooseMedia', { tempFiles: [{ tempFilePath: 'e2e-photo.jpg', fileType: 'image', size: 150000 }], type: 'image', errMsg: 'chooseMedia:ok' });
  await (await l.waitUntil(() => p.$('.photo.add'), 'add photo')).tap();
  d = await l.waitData(p, (x) => x.photos.length > 0, 'photo shows on Me', 20000).catch((e) => ({ err: e.message }));
  await m.restoreWxMethod('chooseMedia');
  l.check(!d.err, 'the upload reaches the server and shows on Me', d.err);
  const status = await l.waitUntil(async () => {
    const list = (await l.api(token, 'GET', '/v1/photos')).data.photos || [];
    return list[0] && list[0].status !== 'pending' ? list[0] : null;
  }, 'photo review to finish', 120000).catch((e) => ({ status: `timed out: ${e.message}` }));
  l.note(`Real photo review said: ${status.status} ${status.reason || ''}`);
  await l.sleep(500);
  await l.shot('me-photo');
};
