# Google Play listing

Paste-ready text for Play Console. Images are in `store/images/play`.

## App details

- **App name:** SynAgent
- **Default language:** English (United States). Add Chinese (Simplified) below.
- **App or game:** App
- **Free or paid:** Free, with in-app purchases (credits)
- **Category:** Dating
- **Contact email:** support@heros-agent.space
- **Website:** https://match.heros-agent.space
- **Privacy policy:** https://match.heros-agent.space/legal/privacy
- **Account deletion URL:** https://match.heros-agent.space/legal/delete-account

## English

**Short description (80 characters at most)**

A matchmaker you talk to, not a deck you swipe. Meet Suki, your AI wingwoman.

**Full description**

SynAgent is a dating app built around Suki, an AI matchmaker you get to know by talking to her. No long forms and no endless swiping.

GET TO KNOW SUKI
Set up your profile in conversation: who you are, your photos, your interests and what you're looking for. She can help write your bio, and nothing is saved to it until you approve.

GOOD INTRODUCTIONS
Suki looks for people in your city who suit you and tells you why she thinks you'd get along. It's only a match when you both say yes.

NEVER STUCK FOR WORDS
Chat with your match directly. Suki can suggest an opener or a question when things go quiet, and only you see her ideas. You can turn coaching off in Settings.

A DATE YOU'LL BOTH ENJOY
She shortlists public places around what you have in common. You each pick a favorite privately, choose a time, and the date is set once you both confirm.

SHE LEARNS YOUR TYPE
Tell her how the date went. What you accept, pass on and say afterwards helps her make better introductions over time.

YOU STAY IN CONTROL
Choose how Suki talks to you, read and delete what she remembers, reset what she has learned, block or report anyone, export your data, or delete your account.

SAFETY
SynAgent is for adults 18 and over. Explicit photos, messages and profile text are filtered. Suki is an AI and can make mistakes. We don't run background checks, so please meet in public and tell someone you trust where you're going.

Your messages, profile and photos are processed by an AI provider, including in China. See the privacy policy for details.

## Chinese (Simplified)

**Short description**

一位可以聊天的红娘，不用一张张滑卡片。认识 Suki，你的 AI 僚机。

**Full description**

SynAgent 是一款约会应用，核心是 AI 红娘 Suki。和她聊聊天就能开始，不用填长表格，也不用没完没了地滑卡片。

认识 Suki
在对话中完成资料：你是谁、你的照片、你的兴趣，以及你想找什么样的人。她可以帮你写个人简介，你确认之后才会保存。

靠谱的介绍
Suki 会在你所在的城市寻找合适的人，并告诉你为什么觉得你们合得来。双方都同意才算配对。

不怕冷场
和对方直接聊天。话题断了的时候，Suki 可以给你一句开场白或一个问题，这些建议只有你能看到。你可以在设置中关闭。

你们都喜欢的约会
她会根据你们的共同点挑选公共场所。你们各自悄悄选出最喜欢的地方，定好时间，双方确认后约会就定下来了。

她越来越懂你
告诉她约会怎么样。你接受或跳过了谁、事后说了什么，都会帮她在之后做出更好的介绍。

由你做主
你可以选择 Suki 和你说话的方式，查看并删除她记住的内容，重置她学到的东西，屏蔽或举报任何人，导出你的数据，或删除账户。

安全
SynAgent 仅限 18 岁及以上的成年人使用。露骨的照片、消息和资料文字会被过滤。Suki 是 AI，可能会出错。我们不做背景调查，所以请在公共场所见面，并告诉你信任的人你要去哪里。

你的消息、资料和照片会由 AI 服务商处理，其中包括位于中国的服务商。详情请见隐私政策。

## App content answers

- **Ads:** no ads.
- **App access:** login is required. Create a test account for reviewers and give them its details in Play Console. A new account can use Suki, but it needs a complete profile with a photo to reach matching.
- **Target audience:** 18 and over only.
- **Content rating:** complete the questionnaire as a dating app with user-generated content and user-to-user chat, no location sharing with others.
- **News, government, financial, health:** no.
- **In-app purchases:** yes. One-time consumable credit packs, sold through Google Play in this build. Product IDs and credit amounts are in `internal/billing/google.go`. Prices are set in Play Console.

## Data safety answers

Collected, not sold. All data is encrypted in transit. Users can request deletion.

| Data type | Collected | Shared | Purpose |
|---|---|---|---|
| Name, email address, user IDs | Yes | No | Account management, app functionality |
| Approximate location (typed city, no GPS) | Yes | Yes, service providers | App functionality |
| Personal info: birthdate, gender, sexual orientation (who they seek), religion, political views | Yes | Yes, AI provider | App functionality, personalization |
| Photos | Yes | Yes, AI provider for moderation | App functionality, safety |
| In-app messages | Yes | Yes, AI provider | App functionality, safety |
| Other user-generated content (bio, prompts, date feedback) | Yes | Yes, AI provider | App functionality, personalization |
| App interactions (introductions accepted or passed) | Yes | No | Personalization |
| Purchase history (credit packs) | Yes | No | App functionality |
| Device or other IDs (push token) | Yes | Yes, Expo and Google for delivery | Notifications |

The shared rows go to service providers; check Play's current definition of "shared", which excludes some service-provider transfers, when answering. Keep these answers in step with the privacy policy.
