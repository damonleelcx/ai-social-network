# SynAgent WeChat mini program

A native mini program that talks to the same server as the app. React Native can't run in WeChat, so these are separate pages. The account, Suki, matches, chats and credits are all shared with the app.

**In it:** WeChat sign-in, plus email sign-in, sign-up, verification and forgot password; Suki's conversation and cards, with her voice (speaker buttons and reading new messages aloud); matches and chat, with report and block; a person's full profile, where tapping any photo shows all of it; date plans and your buddy posts; photos, credits balance and history; notifications; settings for the app, your account (password, sessions, data export, legal, sign-out, deletion), blocked people, Suki and what she remembers. Chinese by default, plus English, Korean, Japanese and Spanish, following the account's language setting. Words live in `lib/catalog.js`, keyed by their English like the app's catalog.

**Not in it yet:** editing your profile (telling Suki works), pausing introductions, push notifications, the coaching hints inside chats, Suki's avatar moods and a light theme. The settings for coaching apply here anyway because they control what Suki reads, and the app shows the hints. The theme and avatar moods settings are left out because they'd change nothing here. Credits can be bought with WeChat Pay, except on iOS (below).

**Check first on a real phone:** saving settings uses `PATCH /v1/settings`, and PATCH isn't among the methods WeChat's `wx.request` documents. Reading settings works either way. If a phone refuses it, the switch goes back and the person is told it didn't save (see `patch` in `lib/api.js`); the fix would be a POST route on the server. Sending the data export uses `wx.shareFileMessage`, which also needs trying on a phone.

## Running it

Open this folder in WeChat DevTools. `project.config.json` uses the tourist AppID, so it runs in the simulator without a real one. Turn off domain checking in DevTools' local settings to reach the server before the domains below are configured.

## What it needs before release

These need the owner's accounts and can't be done from the repo:

1. **Register the mini program** at mp.weixin.qq.com and put the AppID in `project.config.json`.
2. **ICP filing.** WeChat only allows ICP-filed domains on its server lists. The server is in us-east-1, so this probably means a filed domain with a proxy hosted in China. Set the domain in `config.js`.
3. **Sign-in.** Add `SYN_WECHAT_APP_ID` and `SYN_WECHAT_APP_SECRET` to `synagent/prod`. WeChat sign-in stays off until they exist. Someone who first used email and then signs in with WeChat gets a separate account, because WeChat gives no address to match on.
3. **Server domains.** Add that domain under request, socket, downloadFile and uploadFile.
4. **Category and qualifications.** Social and dating categories are restricted, and an AI assistant may need generative-AI filing. Check the current rules for the category you choose before building on it.
5. **WeChat Pay.** Get a merchant account, bind it to this AppID, and enable JSAPI payments. In the merchant console, use the WeChat Pay public key mode. Add these to `synagent/prod` (all of them, or it stays off):
   `SYN_WECHAT_APP_ID`, `SYN_WECHAT_APP_SECRET`, `SYN_WECHAT_MCH_ID`, `SYN_WECHAT_MCH_SERIAL`, `SYN_WECHAT_API_V3_KEY`, `SYN_WECHAT_MCH_PRIVATE_KEY`, `SYN_WECHAT_PAY_PUBLIC_KEY_ID`, `SYN_WECHAT_PAY_PUBLIC_KEY`.
   Payment notifications go to `/v1/billing/wechat` on the server's `PublicOrigin`, which must be an HTTPS address WeChat can reach.
   **iOS:** WeChat doesn't allow selling virtual goods there, so the buttons are hidden on iOS and nothing points elsewhere to pay. iOS users can spend credits they already have. Check the current rules before submitting, because this is what review will look at.
6. **Upload and review** from DevTools.
