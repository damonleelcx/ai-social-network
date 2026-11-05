# SynAgent WeChat mini program

A native mini program that talks to the same server as the app. React Native can't run in WeChat, so these are separate pages. The account, Suki, matches, chats and credits are all shared with the app.

**In it:** WeChat sign-in, plus email sign-in, sign-up, verification and forgot password; Suki's conversation and cards, with her voice (speaker buttons and reading new messages aloud); matches and chat, with report, block and Suki's private coaching hints; a person's full profile, where tapping any photo shows all of it; date plans and your buddy posts; editing your profile and pausing introductions; photos, credits balance and history; a notifications list; settings for the app, your account (password, sessions, data export, legal, sign-out, deletion), blocked people, Suki and what she remembers. Chinese by default, plus English, Korean, Japanese and Spanish, following the account's language setting. Words live in `lib/catalog.js`, keyed by their English like the app's catalog.

**Not in it yet:** push notifications (see "Push notifications" below), Suki's avatar moods and a light theme. The theme and avatar moods settings are left out because they'd change nothing here. Credits can be bought with WeChat Pay, except on iOS, and that needs a decision before release (see item 6 below).

`wx.request` doesn't document PATCH, so the mini program saves settings and the profile with `POST /v1/settings` and `POST /v1/profile`, which the server handles exactly as it does PATCH.

Nothing here has been run in WeChat DevTools or on a phone yet. Try data export there in particular, because it uses `wx.shareFileMessage`.

## Running it

Open this folder in WeChat DevTools. `project.config.json` uses the tourist AppID, so it runs in the simulator without a real one. Turn off domain checking in DevTools' local settings to reach the server before the domains below are configured.

## What it needs before release

These need the owner's accounts and can't be done from the repo:

1. **Register the mini program** at mp.weixin.qq.com and put the AppID in `project.config.json`.
2. **ICP filing.** WeChat only allows ICP-filed domains on its server lists. The server is in us-east-1, so this probably means a filed domain with a proxy hosted in China. Set the domain in `config.js`.
3. **Sign-in.** Add `SYN_WECHAT_APP_ID` and `SYN_WECHAT_APP_SECRET` to `synagent/prod`. WeChat sign-in stays off until they exist. Someone who first used email and then signs in with WeChat gets a separate account, because WeChat gives no address to match on.
4. **Server domains.** Add that domain under request, socket, downloadFile and uploadFile.
5. **Category and qualifications.** Suki's messages and coaching hints carry an "AI-generated" label, which the AI categories ask for. Social and dating categories are restricted, and an AI assistant may need generative-AI filing. Check the current rules for the category you choose before building on it.
6. **Payments: open question for the owner.** WeChat's [operating rules](https://developers.weixin.qq.com/miniprogram/product/) (sections 2.5.13 and 2.14 item 7, page last edited 2026-11-03) say virtual goods, which they say include virtual currency, must be bought through WeChat's mini program virtual payment. They also forbid sending people to an app, official account, H5 page, personal account or website to pay, and 2.14 adds Android payment. Three things follow:
   - **No pointing elsewhere.** The mini program doesn't, and mustn't. When it can't sell credits, it says only that they can't be bought here right now and that existing credits still work.
   - **WeChat Pay may be the wrong channel.** Credits look like virtual currency, but the code below uses ordinary WeChat Pay (JSAPI). An [official guide](https://developers.weixin.qq.com/community/minihome/doc/00002cf077cd4810fee42f4b865c01) says ordinary WeChat Pay gets switched off for virtual-only categories on Android and other non-iOS systems. Confirm this with WeChat. Moving to virtual payment means new server and mini program work, which hasn't been started.
   - **iOS may have changed.** Hiding the buy buttons on iOS was built on the older understanding that virtual goods couldn't be sold there. WeChat's community posts now say virtual payment is open on iOS. Behaviour is unchanged for now: iOS users see no buy buttons and can spend credits they already have.

   Until this is settled, people who run out of credits and can't buy (iOS, or WeChat Pay off) can't keep talking to Suki, though chats with matches stay free. Suki's own out-of-credits line comes from the server and still says "Top up", which doesn't fit those people.
7. **WeChat Pay, if it's kept.** Get a merchant account, bind it to this AppID, and enable JSAPI payments. In the merchant console, use the WeChat Pay public key mode. Add these to `synagent/prod` (all of them, or it stays off):
   `SYN_WECHAT_APP_ID`, `SYN_WECHAT_APP_SECRET`, `SYN_WECHAT_MCH_ID`, `SYN_WECHAT_MCH_SERIAL`, `SYN_WECHAT_API_V3_KEY`, `SYN_WECHAT_MCH_PRIVATE_KEY`, `SYN_WECHAT_PAY_PUBLIC_KEY_ID`, `SYN_WECHAT_PAY_PUBLIC_KEY`.
   Payment notifications go to `/v1/billing/wechat` on the server's `PublicOrigin`, which must be an HTTPS address WeChat can reach.
8. **Push notifications.** None are sent to WeChat today, and nothing here pretends otherwise. They need WeChat [subscription messages](https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/subscribe-message.html):
   - **From the owner:** the registered AppID (item 1), and subscription message templates chosen or applied for at mp.weixin.qq.com. Templates are needed for at least new introductions and matches, new chat messages, date updates and messages from Suki. Each gives a template ID and its own fields.
   - **In the mini program:** call `wx.requestSubscribeMessage` with those template IDs from a tap, and send the result to the server. People usually grant one send at a time, so it needs asking again. DevTools can't test it, so it needs a real phone.
   - **On the server:** store each person's openid and what they've granted, and send through WeChat's `subscribeMessage.send` using an access token from the AppID and secret. It should follow the existing notify settings, and needs a way to record WeChat's delivery results. The push code handles Expo tokens today, not WeChat.
9. **Upload and review** from DevTools.
