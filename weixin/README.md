# SynAgent WeChat mini program

A native mini program that talks to the same server as the app. React Native can't run in WeChat, so these are separate pages. The account, Suki, matches, chats and credits are all shared with the app.

**In it:** WeChat sign-in, plus email sign-in, sign-up, verification and forgot password; Suki's conversation and cards, with her voice (speaker buttons and reading new messages aloud); matches and chat, with report, block and Suki's private coaching hints; a person's full profile, where tapping any photo shows all of it; date plans and your buddy posts; editing your profile and pausing introductions; photos, credits balance and history; a daily check-in (签到) on Me; sharing to chats and Moments from Suki's page, and an invite card on Me; a notifications list; settings for the app, your account (password, sessions, data export, legal, sign-out, deletion), blocked people, Suki and what she remembers. Chinese by default, plus English, Korean, Japanese and Spanish, following the account's language setting. Words live in `lib/catalog.js`, keyed by their English like the app's catalog.

**Not in it yet:** push notifications (see "Push notifications" below) and a light theme. The theme setting is left out because it would change nothing here. Credits can be bought with WeChat Pay, except on iOS, and that needs a decision before release (see item 6 below).

`wx.request` doesn't document PATCH, so the mini program saves settings and the profile with `POST /v1/settings` and `POST /v1/profile`, which the server handles exactly as it does PATCH.

It has been run end to end in WeChat DevTools (see `e2e/README.md`), but not on a phone yet. On a phone, try sending the data export, the photo picker, the keyboard with the coaching panel, and hearing Suki's voice.

## Running it

Open this folder in WeChat DevTools. `project.config.json` uses the tourist AppID, so it runs in the simulator without a real one. Turn off domain checking in DevTools' local settings to reach the server before the domains below are configured.

## Sharing, invites and check-in

Shares carry only the sharer's opaque invite code (`ref=`), and always a fixed image from `assets/share` (made by `e2e/make_assets.py`), since WeChat otherwise uses a screenshot of the page. Nothing a share says, and nothing next to the share button, mentions credits or any reward: WeChat's rules forbid rewarding or promising a reward for sharing (诱导分享). Credits are paid by the server only when an invited friend's *new* account goes live, to both of them, within per-day and lifetime caps, and the daily check-in pays once per China calendar day. Amounts and caps are server settings: `SYN_INVITE_CREDITS_INVITER`, `SYN_INVITE_CREDITS_INVITEE`, `SYN_INVITE_DAILY_CAP`, `SYN_INVITE_TOTAL_CAP`, `SYN_CHECKIN_CREDITS`.

Only Suki's page offers Moments: Moments opens the shared page by itself, without sign-in, and her page then shows a hello instead of anyone's data.

## What it needs before release

These need the owner's accounts and can't be done from the repo:

1. **Register the mini program** at mp.weixin.qq.com and put the AppID in `project.config.json`.
2. **ICP filing.** WeChat only allows ICP-filed domains on its server lists. The server is in us-east-1, so this probably means a filed domain with a proxy hosted in China. Set the domain in `config.js`.
3. **Sign-in.** Add `SYN_WECHAT_APP_ID` and `SYN_WECHAT_APP_SECRET` to `synagent/prod`. WeChat sign-in stays off until they exist. Someone who first used email and then signs in with WeChat gets a separate account, because WeChat gives no address to match on.
4. **Server domains.** Add that domain under request, socket, downloadFile and uploadFile.
5. **Category and qualifications.** Suki's messages and coaching hints carry an "AI-generated" label, which the AI categories ask for. Social and dating categories are restricted, and an AI assistant may need generative-AI filing. Check the current rules for the category you choose before building on it.
6. **Payments: virtual payment (小程序虚拟支付).** Credits are virtual goods. On 2026-11-06 WeChat closed ordinary WeChat Pay for this mini program on Android and other non-iOS systems ("小程序对应支付能力已被限制"), so credits are sold through virtual payment, buying an item directly (`short_series_goods`). The mini program never points anywhere else to pay. Setup, in the HelloSuki console:
   - **Open it:** 支付与交易 → 虚拟支付 (needs the 认证 company account and a linked merchant). Note the **OfferID** and the **AppKey**: the sandbox key for testing, the live key for production.
   - **Items (道具):** one per pack, with these ids and prices in fen: `credits_100` 3500, `credits_300` 8800, `credits_1000` 23800. A price that differs from `WeChatPackages` in `internal/billing/wechat.go` makes WeChat refuse the order.
   - **Delivery push:** in 开发管理 → 消息推送, set the URL to `https://suki.zhiyuwujie.top/v1/billing/wechat/push` with JSON format. Safe mode needs the EncodingAESKey, too. Then turn on the 道具发货推送 (xpay_goods_deliver_notify) in the virtual payment settings.
   - **Server settings** (the China node's `synagent-wechat-xpay` secret; all or none): `SYN_WECHAT_XPAY_OFFER_ID`, `SYN_WECHAT_XPAY_APP_KEY`, `SYN_WECHAT_XPAY_ENV` (`1` sandbox, `0` live, and the AppKey must match), `SYN_WECHAT_PUSH_TOKEN`, and `SYN_WECHAT_PUSH_AES_KEY` for safe mode. The AppID and secret come from `SYN_WECHAT_APP_ID` and `SYN_WECHAT_APP_SECRET`.
   - **How it works:** the server signs each order (`paySig` with the AppKey, `signature` with the session_key from a fresh `wx.login`). The delivery push adds the credits once. If the push is late, the mini program asks the server to check the order (`/xpay/query_order`), which adds the credits and reports the delivery (`/xpay/notify_provide_goods`).
   - **iOS:** with virtual payment on, iOS shows the packs too. Confirm on a real iPhone before relying on it.
7. **Ordinary WeChat Pay (JSAPI), the fallback.** It's used only while virtual payment isn't configured. Get a merchant account, bind it to this AppID, and enable JSAPI payments. In the merchant console, use the WeChat Pay public key mode. Add these to `synagent/prod` (all of them, or it stays off):
   `SYN_WECHAT_APP_ID`, `SYN_WECHAT_APP_SECRET`, `SYN_WECHAT_MCH_ID`, `SYN_WECHAT_MCH_SERIAL`, `SYN_WECHAT_API_V3_KEY`, `SYN_WECHAT_MCH_PRIVATE_KEY`, `SYN_WECHAT_PAY_PUBLIC_KEY_ID`, `SYN_WECHAT_PAY_PUBLIC_KEY`.
   Payment notifications go to `/v1/billing/wechat` on the server's `PublicOrigin`, which must be an HTTPS address WeChat can reach.
8. **Push notifications.** None are sent to WeChat today, and nothing here pretends otherwise. They need WeChat [subscription messages](https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/subscribe-message.html):
   - **From the owner:** the registered AppID (item 1), and subscription message templates chosen or applied for at mp.weixin.qq.com. Templates are needed for at least new introductions and matches, new chat messages, date updates and messages from Suki. Each gives a template ID and its own fields.
   - **In the mini program:** call `wx.requestSubscribeMessage` with those template IDs from a tap, and send the result to the server. People usually grant one send at a time, so it needs asking again. DevTools can't test it, so it needs a real phone.
   - **On the server:** store each person's openid and what they've granted, and send through WeChat's `subscribeMessage.send` using an access token from the AppID and secret. It should follow the existing notify settings, and needs a way to record WeChat's delivery results. The push code handles Expo tokens today, not WeChat.
9. **Upload and review** from DevTools.
