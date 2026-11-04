# SynAgent WeChat mini program

A native mini program that talks to the same server as the app. React Native can't run in WeChat, so these are separate pages. The account, Suki, matches, chats and credits are all shared with the app.

**In it:** sign in, sign up and email verification; Suki's conversation and cards; matches and chat, with report and block; date plans and your buddy posts; photos, credits balance and history; account deletion. Chinese only.

**Not in it yet:** buying credits, voice, notifications and settings.

## Running it

Open this folder in WeChat DevTools. `project.config.json` uses the tourist AppID, so it runs in the simulator without a real one. Turn off domain checking in DevTools' local settings to reach the server before the domains below are configured.

## What it needs before release

These need the owner's accounts and can't be done from the repo:

1. **Register the mini program** at mp.weixin.qq.com and put the AppID in `project.config.json`.
2. **ICP filing.** WeChat only allows ICP-filed domains on its server lists. The server is in us-east-1, so this probably means a filed domain with a proxy hosted in China. Set the domain in `config.js`.
3. **Server domains.** Add that domain under request, socket, downloadFile and uploadFile.
4. **Category and qualifications.** Social and dating categories are restricted, and an AI assistant may need generative-AI filing. Check the current rules for the category you choose before building on it.
5. **Payment.** Stripe isn't usable here. Selling credits needs WeChat Pay, and virtual goods are restricted on iOS. The mini program shows the balance and doesn't send people elsewhere to pay.
6. **Upload and review** from DevTools.
