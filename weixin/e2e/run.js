// node run.js [flow ...]  Runs the named flows from flows/, or every one in order.
const fs = require('fs');
const path = require('path');
const lib = require('./lib');

(async () => {
  const dir = path.join(__dirname, 'flows');
  const all = fs.readdirSync(dir).filter((f) => f.endsWith('.js')).sort().map((f) => f.replace(/\.js$/, ''));
  const wanted = process.argv.slice(2);
  const names = wanted.length ? all.filter((n) => wanted.some((w) => n.includes(w))) : all;
  await lib.connect();
  for (const name of names) {
    console.log(`\n=== ${name}`);
    try {
      await require(path.join(dir, name))(lib);
    } catch (e) {
      lib.check(false, `${name} stopped`, e.stack || e.message);
    }
  }
  const bad = lib.summary();
  lib.m().disconnect();
  process.exit(bad ? 1 : 0);
})();

setTimeout(() => {
  console.error('Gave up after the time limit.');
  lib.summary();
  process.exit(2);
}, Number(process.env.WX_LIMIT_MS || 540000));
