const api = require('./lib/api');
const live = require('./lib/live');
const share = require('./lib/share');

App({
  onLaunch(options) {
    share.remember(options.query);
  },
  onShow(options) {
    share.remember(options.query);
    if (api.getToken()) live.connect();
  },
  onHide() {
    live.disconnect();
  },
});
