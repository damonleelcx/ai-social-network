const api = require('./lib/api');
const live = require('./lib/live');

App({
  onShow() {
    if (api.getToken()) live.connect();
  },
  onHide() {
    live.disconnect();
  },
});
