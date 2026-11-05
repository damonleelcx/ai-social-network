# SynAgent API

The contract between the backend and the mobile app. JSON over HTTPS.

- Auth: `Authorization: Bearer <token>`. The token comes from sign-up or sign-in.
- Errors: non-2xx with `{"error": "message safe to show the person"}`.
- `401`: signed out. `403` "please verify your email first": send them to the verify screen.
- **No endpoint returns a score, rank or evaluation.**

## Live updates

`GET /v1/ws` is a WebSocket with the same `Authorization` header (React Native:
`new WebSocket(url, null, {headers})`). It sends `{"topic": "..."}` meaning "something
changed, refetch". Reconnect with backoff, and refetch everything on reconnect.

| Topic | Refetch |
|---|---|
| `agent` | `GET /v1/agent/messages?after=<last seq>` |
| `typing:agent` | show Suki typing (clear on `agent`, or after 30s) |
| `proposals` | `GET /v1/proposals` |
| `matches` | `GET /v1/matches` |
| `chat:<match id>` | that chat's messages and hints; the match list |
| `typing:<match id>` | show the other person typing for 4s |
| `dates` | `GET /v1/dates` and any open date |
| `photos` | `GET /v1/photos`, `GET /v1/me` |
| `notification` | `GET /v1/notifications` |

Suki's replies also stream: `{"topic":"agent_stream","data":{"turn","offset","text","reset"}}`. Append `text` at `offset` (in UTF-16 units) for that turn; `reset:true` means drop what's shown. This is a preview: the `agent` topic and the stored message replace it, and `meta` arrives only then. Text is checked by the score-leak and NSFW guards before release, so it runs a few words behind the model.

## Accounts

| Call | Body | Returns |
|---|---|---|
| `POST /v1/auth/signup` | `email, password, name, accept_terms, language?` | `201 {token, expires_at, user}` and emails a 6-digit code |
| `POST /v1/auth/signin` | `email, password` | `{token, expires_at, user}` |
| `POST /v1/auth/signout` | | `{ok}` |
| `POST /v1/auth/verify` | `code` | `{user}` (signed in) |
| `POST /v1/auth/resend` | | `{ok}` (signed in) |
| `POST /v1/auth/forgot` | `email` | always `{ok}` |
| `POST /v1/auth/reset` | `email, code, password` | `{ok}`; all sessions are ended, so sign in again |
| `POST /v1/auth/change-password` | `current, next` | `{ok}` |
| `GET /v1/auth/sessions` | | `{sessions:[{id,user_agent,ip,created_at,last_seen_at,current}]}` |
| `DELETE /v1/auth/sessions/{id}` | | `{ok}` |
| `POST /v1/account/delete` | `password` | `{ok}` |
| `GET /v1/account/export` | | a JSON file of their data |
| `GET /v1/me` | | `{user, profile?, missing?}` |

`user`: `{id, email, name, role, email_verified, created_at, admin}`. A password has at least 10 characters.

Public pages (open in a browser): `/`, `/legal/terms`, `/legal/privacy`, `/legal/delete-account`.

## Profile and photos

`GET /v1/profile` returns `{profile, missing:[...]}`. `PATCH /v1/profile` with any subset returns `{profile}`. `POST /v1/profile` does the same, for clients with no PATCH (WeChat's `wx.request`):

| Field | Values |
|---|---|
| `display_name, pronouns, bio, occupation, education, religion, politics` | text |
| `birthdate` | `yyyy-mm-dd`; set once; under 18 returns 403 |
| `gender` | `woman man nonbinary other` |
| `seeking` | list of the same |
| `age_min, age_max` | 18 to 120 |
| `goal` | `long_term marriage short_term casual friends_first not_sure` |
| `height_cm` | 120 to 230 |
| `smoking, drinking` | `no sometimes yes` |
| `kids` | `none have` |
| `wants_kids` | `yes no open` |
| `languages, interests` | lists of text |
| `prompts` | `[{q, a}]`, up to 6 |
| `dealbreakers` | `{field: [values]}` for smoking, drinking, kids, wants_kids, religion, politics |

A profile field filtered as NSFW returns `422` with a message to show. Read-only: `user_id, age, city_key, city_name, status` (`onboarding active paused held`), `bio_draft`.

| Call | Body | Returns |
|---|---|---|
| `POST /v1/profile/city` | `query` | `{city_name}` |
| `POST /v1/profile/pause` | `paused` | `{paused}` |
| `POST /v1/profile/bio-draft/accept` | | `{ok}` |
| `GET /v1/photos` | | `{photos:[{id,width,height,status,reason,tips,is_primary}]}`; status is `pending approved rejected review` |
| `POST /v1/photos` | raw JPEG, PNG or WebP bytes, up to 12 MB (`Content-Type: image/jpeg`) | `201 {id, status}` |
| `POST /v1/photos/{id}/primary` | | `{ok}` |
| `DELETE /v1/photos/{id}` | | `{ok}` |
| `GET /v1/media/{id}` | | JPEG; needs the auth header, so pass headers to the image component |

Up to 6 photos. Review is asynchronous; watch the `photos` topic.

## Suki

| Call | Body | Returns |
|---|---|---|
| `GET /v1/agent/messages?after=<seq>` | | `{messages:[Message], thinking:bool}` (the last 100 without `after`). `after` responses may overlap slightly, here and in chats, so merge by `seq` |
| `POST /v1/agent/messages` | `text, client_id` (a uuid per message, safe to retry) | `202 {message}` |

The answer arrives later: watch `agent`. `thinking` is true while a reply is pending.

`Message`: `{thread_id, seq, sender_id?, role, body, meta?, read_at?, created_at}`. `role` is `user`, `assistant` or `event`.

`meta` on a Suki message: `mood` (`happy surprised thinking playful pouty`) picks the avatar image; `replies` is a list of quick replies, shown only on the newest message; `cards` is a list of:

| Card `kind` | Fields | Render as |
|---|---|---|
| `photo_upload` | | Button: pick a photo and `POST /v1/photos` |
| `bio_draft` | `text` | The text, with "Use this" (`POST .../bio-draft/accept`) |
| `proposal` | `proposal_id` | Person card: fetch `GET /v1/proposals/{id}`, then Yes or Pass |
| `match` | `match_id` | Button: open the chat |
| `date` | `date_id`, `focus?` (`time`, `change`) | Button: open the date screen |
| `share_plan` | `date_id` | Button: `GET /v1/dates/{id}/share`, then the OS share sheet |
| `open_screen` | `screen`, `label` | Button. Screens: `settings settings_app settings_agent settings_account settings_memory settings_blocked profile_edit photos matches dates verify_email` |
| `confirm` | `action`, `label`, plus `photo_id`, `match_id` or `user_id` | Button with an "are you sure" step. Actions: `remove_photo`, `unmatch`, `block`, `reset_taste` |

Suki's voice: `GET /v1/agent/speech` returns `{enabled}`. `POST /v1/agent/speech` with `{seq}` (one of her messages) or `{sample:true}` returns `audio/mpeg`. `429` means the daily allowance is used.

## Introductions and matches

| Call | Body | Returns |
|---|---|---|
| `GET /v1/proposals` | | `{proposals:[Proposal]}` waiting for this person |
| `GET /v1/proposals/{id}` | | `{proposal}` |
| `POST /v1/proposals/{id}/decide` | `decision`: `accept` or `pass` | `{match_id}`; empty until both have said yes |
| `GET /v1/matches` | | `{matches:[Match]}` |
| `GET /v1/matches/{id}` | | `{match}` |
| `POST /v1/matches/{id}/unmatch` | | `{ok}` |
| `GET /v1/matches/{id}/messages?after=<seq>` | | `{messages}` |
| `POST /v1/matches/{id}/messages` | `text, client_id` | `201 {message}`; `422` with a message to show if it's filtered as NSFW |
| `POST /v1/matches/{id}/read` | `seq` | `{ok}` |
| `POST /v1/matches/{id}/typing` | | `{ok}` |
| `GET /v1/matches/{id}/hints` | | `{hints:[{id,body,suggestions,created_at}]}`, private to this person |
| `POST /v1/matches/{id}/coach` | | `202`; Suki answers with a new hint |
| `POST /v1/hints/{id}/dismiss` | | `{ok}` |

`Proposal`: `{id, status, pitch, shared:[...], my_decision, person, created_at, expires_at, match_id?}`

`Match`: `{id, stage, status, person, created_at, last_message_at, last_preview, unread, open_date_id?}`. `stage`: `icebreaker getting_to_know date_ready planning planned post_date`.

`person`: `{user_id, display_name, age, gender, pronouns, city_name, goal, bio, occupation, education, height_cm, smoking, drinking, kids, wants_kids, languages, interests, prompts, photo_ids}`.

A chat `Message`'s `role` is `text` (`sender_id` is a person), or from Suki with no `sender_id`: `prompt` (a shared question), `date` (a date update, `meta.date_id`, and `meta.card` of `vote`, `time`, `confirm` or `confirmed`), or `system`. Tapping a hint suggestion fills the input box; it never sends.

## Dates

| Call | Body | Returns |
|---|---|---|
| `GET /v1/dates` | | `{dates:[DatePlan]}` |
| `POST /v1/matches/{id}/date` | `constraints?:[...]` | `201 {date_id}` |
| `GET /v1/dates/{id}` | | `{date}` |
| `POST /v1/dates/{id}/vote` | `venue_id` | `{date}` |
| `POST /v1/dates/{id}/time` | `at` (RFC 3339) | `{date}`; also reschedules |
| `POST /v1/dates/{id}/confirm` | | `{date}`; only the person who didn't propose |
| `POST /v1/dates/{id}/cancel` | `reason?` | `{date}` |
| `GET /v1/dates/{id}/share` | | `{text}` |

`DatePlan`: `{id, match_id, status, venues:[Venue], my_vote, other_voted, venue, constraints, proposed_at, proposed_by_me, scheduled_at, person, created_at}`. `status`: `searching voting scheduling confirmed completed cancelled failed`.

`Venue`: `{id, name, kind, address, lat, lon, website?, hours?, reason?}`.

## Safety

| Call | Body | Returns |
|---|---|---|
| `GET /v1/blocks` | | `{blocked:[{user_id,display_name}]}` |
| `POST /v1/blocks` | `user_id` | `{ok}`; ends the match at once |
| `DELETE /v1/blocks/{id}` | | `{ok}` |
| `POST /v1/reports` | `user_id, match_id?, message_seq?, reason, details?` | `201` |

Reasons: `harassment`, `fake_profile`, `inappropriate_content`, `scam`, `underage`, `other`.

## Settings and memory

`GET /v1/settings` returns `{settings}`. `PATCH /v1/settings` with any subset returns `{settings}`. `POST /v1/settings` does the same, for clients with no PATCH (WeChat's `wx.request`).

| Key | Values | Default |
|---|---|---|
| `language` | `en zh ko ja es` | `en` |
| `theme` | `system light dark` | `system` |
| `notify_matches, notify_messages, notify_dates, notify_suki, notify_email` | bool | true |
| `read_receipts` | bool | true |
| `daily_matches` | 1 to 5 | 3 |
| `suki_tone` | `playful warm direct calm` | `playful` |
| `suki_emoji` | `none some lots` | `some` |
| `suki_proactive` | `low normal high` | `normal` |
| `suki_nickname` | text | `""` |
| `mood_avatar` | bool | true |
| `suki_voice` | bool: show speaker buttons | true |
| `suki_voice_auto` | bool: read new messages aloud | false |
| `chat_coach` | bool: Suki gives me hints | true |
| `chat_coach_read` | bool: Suki may read my messages to coach | true |
| `chat_prompts` | bool: Suki may post questions in chats | true |
| `memory_enabled` | bool | true |
| `debrief_enabled` | bool | true |

| Call | Returns |
|---|---|
| `GET /v1/memories` | `{memories:[{id,note,created_at}]}` |
| `DELETE /v1/memories/{id}` | `{ok}` |
| `POST /v1/memories/reset` | `{ok}`: forgets notes and everything learned |

## Notifications

| Call | Body | Returns |
|---|---|---|
| `GET /v1/notifications` | | `{notifications:[{id,kind,title,body,data,read,created_at}]}` |
| `POST /v1/notifications/read` | | `{ok}` |
| `POST /v1/push-tokens` | `token` (`ExponentPushToken[...]`), `platform` | `{ok}` |

`data.screen` is `agent`, `proposal`, `chat` or `date`, with `data.id`.
