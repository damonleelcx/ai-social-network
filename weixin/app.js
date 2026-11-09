const api = require('./lib/api');
const live = require('./lib/live');
const settings = require('./lib/settings');
const share = require('./lib/share');
const voice = require('./lib/voice');

App({
  onLaunch(options) {
    share.remember(options.query);
  },
  onShow(options) {
    share.remember(options.query);
    if (!api.getToken()) return;
    live.connect();
    api.suki.opened().catch(() => {});
    // Picks up their language. Until it arrives, the last one on this phone stays.
    settings.load().catch(() => {});
  },
  onHide() {
    live.disconnect();
    voice.stop();
  },
});
