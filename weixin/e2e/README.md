# Mini program end-to-end tests

These drive the real mini program in WeChat DevTools, against a server on this machine: real taps, real pages, real model. They refuse any server or DevTools that isn't local, and they hold no secrets.

## Before running

1. **A local server** on `http://127.0.0.1:8080`, using a database you can throw away. Payments off is fine. The flows create accounts and move data around.
2. **A copy of `weixin/`** for DevTools to open, with `config.js` pointing at the local server and domain checking off. Put a photo with a face in the copy as `e2e-photo.jpg`, because `wx.chooseMedia` hands back a path the simulator can read.
3. **DevTools in automation mode:** `cli auto --project <copy> --auto-port 9420 --lang en`
4. **Dependencies:** `npm install` in this folder.

## Running

```
WX_PHOTOS=<folder of photos with faces> node run.js            # every flow, in order
node run.js 02-auth 06-chat                                    # just these
```

| Variable | Default | |
| --- | --- | --- |
| `WX_WS` | `ws://127.0.0.1:9420` | DevTools |
| `WX_API` | `http://127.0.0.1:8080` | The server |
| `WX_PG_CONTAINER`, `WX_PG_DB` | `syn-pg`, `syn_live` | The database, reached through `docker exec` |
| `WX_PHOTOS` | | Photos for the other test people, flow 05 |
| `WX_OUT` | `out/` | Screenshots and the test accounts' details. Ignored by git |
| `WX_LIMIT_MS` | 9 minutes | When the whole run gives up |

Flows build on each other: 02 makes the main account, 04 sets up its profile, and 05 makes the other people and the first match. Flows after that expect those to exist. Run them against a fresh database. Some aren't meant to run twice on the same data, such as the second match in 06.

## What isn't real

- **Native dialogs** can't be tapped, so `wx.showModal`, `wx.showActionSheet`, `wx.chooseMedia`, `wx.previewImage` and `wx.shareFileMessage` are mocked with `mockWxMethod`. The flow says so where it does.
- **Verification codes** aren't mailed locally and are stored hashed, so a test writes the hash of a code it knows into `email_codes`.
- **Cities and places** are seeded in `cached_lookups`, in case the server can't reach OpenStreetMap.
- **Date time** is moved into the past in the database, to get to the debrief without waiting.
- **The out-of-credits card** is added to the page by hand in flow 10, because a server with payments off never charges anyone.
- **Switches and pickers** are changed with `trigger('change')`. A `tap()` doesn't flip them under automation.

## Knowing the tools

- `restoreWxMethod` on a method that isn't mocked wipes the real one until the next compile, so only restore what you mocked. A run that stops early can leave a mock behind; recompile in DevTools to clear it.
- A mock answers by returning or resolving. Wrap the original with `this.origin`, as flow 09 does.
- Custom components are found with `wx-suki-card`, not `suki-card`.
