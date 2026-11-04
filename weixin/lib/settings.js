// The account's settings, shared with the app. Language follows them.
const api = require('./api');
const i18n = require('./i18n');

let cached = null;

function keep(settings) {
  cached = settings;
  i18n.setLanguage(settings.language);
  return settings;
}

module.exports = {
  load: async () => keep(await api.settings.get()),
  patch: async (change) => keep(await api.settings.patch(change)),
  current: () => cached,
  clear() {
    cached = null;
  },
};
