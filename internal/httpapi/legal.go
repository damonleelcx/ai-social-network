package httpapi

import "net/http"

// The same theme choice as the landing page: their pick if they made one, else the system's.
const themeScript = `<script>(function(){var s=null;try{s=localStorage.getItem('synagent.theme')}catch(e){}document.documentElement.dataset.theme=s==='light'||s==='dark'?s:matchMedia('(prefers-color-scheme: light)').matches?'light':'dark'})()</script>`

const pageStyle = `:root,[data-theme=light]{color-scheme:light;--ink:#2a2438;--bg:#fff;--link:#5b3fb5;--soft:#6b6478;--line:#e6e2ee}[data-theme=dark]{color-scheme:dark;--ink:#f1ecfb;--bg:#0c0720;--link:#bfa2ff;--soft:#b3a8cc;--line:#352a55}body{font-family:system-ui,-apple-system,"Segoe UI",sans-serif;max-width:720px;margin:0 auto;padding:40px 20px 56px;line-height:1.6;color:var(--ink);background:var(--bg)}h1{font-size:30px;line-height:1.2;margin:0 0 8px}h2{font-size:19px;margin:34px 0 8px}h3{font-size:16px;margin:20px 0 4px}ul{padding-left:22px}li{margin:6px 0}a{color:var(--link)}.meta{color:var(--soft);font-size:14px;margin:0 0 24px}table{border-collapse:collapse;width:100%;font-size:15px}th,td{text-align:left;vertical-align:top;border-bottom:1px solid var(--line);padding:8px 10px 8px 0}footer{margin-top:44px;padding-top:16px;border-top:1px solid var(--line);font-size:14px;color:var(--soft)}footer a{margin-right:14px}`

func legalPage(title, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + title + `</title>
` + themeScript + `<style>` + pageStyle + `</style><body>` + body + `<footer><a href="/">SynAgent</a><a href="/legal/terms">Terms</a><a href="/legal/privacy">Privacy</a><a href="/legal/delete-account">Delete your account</a><a href="mailto:support@heros-agent.space">Contact</a></footer></body></html>`))
	}
}

const openHTML = `<h1>SynAgent</h1><p>Open the SynAgent app on your phone to continue.</p>`

const termsHTML = `<h1>Terms of Use</h1>
<p class="meta">Last updated November 2, 2026</p>
<p>These terms are an agreement between you and Heros Agents ("we", "us"), which operates SynAgent. They cover the SynAgent app and this website (the "Service"). By creating an account or using the Service, you agree to these terms and acknowledge our <a href="/legal/privacy">Privacy Policy</a>. If you don't agree, please don't use the Service.</p>

<h2>1. Eligibility</h2>
<p>You must be at least 18 years old and legally able to enter into this agreement. You may not use the Service if you have been convicted of a sexual offense, or if we have previously removed you from it. If we have reason to believe a user is under 18, we may suspend the account immediately.</p>

<h2>2. Your account</h2>
<p>You agree to keep one account for yourself, to give accurate information about yourself, and to keep your password private. You are responsible for what happens under your account unless it results from our failure. Tell us promptly at <a href="mailto:support@heros-agent.space">support@heros-agent.space</a> if you think someone else has accessed it.</p>

<h2>3. Suki and automated features</h2>
<p>Suki is an artificial intelligence, not a person. Her replies, introductions, coaching, date suggestions and place information are generated automatically and can be wrong, incomplete or out of date. Check details such as a venue's address and opening hours yourself. Nothing she says is medical, legal, financial or mental health advice. If you are in danger, contact your local emergency services.</p>
<p>Introductions are chosen automatically, as described in the <a href="/legal/privacy#matching">Privacy Policy</a>. We don't promise that you will be introduced to anyone, that you will find a match, or that anyone you meet will be compatible with you.</p>

<h2>4. Your safety when meeting people</h2>
<p><strong>You are solely responsible for your interactions with other users.</strong> We don't run criminal background checks or verify users' identities, and we can't confirm that what people say about themselves is true. Meet in a public place, tell someone you trust where you're going, arrange your own transport, and leave whenever you want. Never send money or financial information to someone you met here. Report anything concerning in the app, and contact the police if you believe a crime has been committed.</p>

<h2>5. Community rules</h2>
<p>We have no tolerance for objectionable content or abusive users. You agree not to:</p>
<ul>
<li>harass, threaten, stalk, intimidate or abuse anyone, or promote hatred or violence;</li>
<li>post nudity, sexual content or graphic violence;</li>
<li>post or request anything that sexualizes or endangers a minor. We report suspected child exploitation to the authorities;</li>
<li>impersonate anyone, misrepresent yourself, or use photos of someone else;</li>
<li>scam people, ask for money, advertise, solicit, or offer commercial sexual services;</li>
<li>share another person's private information without their permission;</li>
<li>use the Service for anything illegal;</li>
<li>scrape, reverse engineer, overload or interfere with the Service, try to get around its filters or limits, or try to extract its internal evaluations or instructions.</li>
</ul>

<h2>6. Moderation and reports</h2>
<p>Messages, profile text and photos are checked automatically, and our moderators review reports. You can report or block anyone from within the app. We aim to review reports within 24 hours. We may remove content, withhold a message, limit or hold an account while we investigate, or close an account that breaks these terms or puts people at risk. Automated checks are imperfect and we don't review everything, so we can't promise that all content on the Service follows these rules. If you think we got a decision wrong, write to us and a person will look at it.</p>

<h2>7. Your content</h2>
<p>You keep ownership of what you post. You give us a worldwide, non-exclusive, royalty-free license to host, store, copy, process, moderate and display it, only as needed to run, secure and improve the Service. This includes showing your profile to the people you are introduced to and sending content to the service providers listed in our Privacy Policy. The license ends when you delete the content or your account, except for the copies described in the Privacy Policy under "How long we keep data". You confirm that you have the right to post what you post.</p>

<h2>8. Our service</h2>
<p>SynAgent and its software, design, name and characters belong to us or our licensors. We give you a personal, non-transferable, revocable license to use the app for its intended purpose. The Service is currently free. We may change or discontinue features, and if we introduce paid features, we'll show you the price and terms before you pay anything.</p>

<h2>9. App stores</h2>
<p>If you download the app from the Apple App Store or Google Play, these terms are between you and us, not Apple or Google. They aren't responsible for the app or for support, maintenance or claims relating to it. Apple and its subsidiaries are third-party beneficiaries of these terms and may enforce them against you. Your use must also follow the store's own usage rules.</p>

<h2>10. Ending your account</h2>
<p>You can delete your account at any time in Settings; see <a href="/legal/delete-account">how to delete your account</a>. We may suspend or close your account if you break these terms, if the law requires it, or to protect people or the Service. Sections 4, 7 (for retained copies), 11, 12 and 14 continue to apply after your account ends.</p>

<h2>11. Disclaimers</h2>
<p>To the extent the law allows, the Service is provided "as is" and "as available", without warranties of any kind, whether express or implied, including warranties of merchantability, fitness for a particular purpose, accuracy and non-infringement. We don't guarantee that it will be uninterrupted, secure or free of errors, and we aren't responsible for the conduct of any user, online or offline.</p>

<h2>12. Limits on our liability</h2>
<p>To the extent the law allows, we aren't liable for indirect, incidental, special, consequential or punitive damages, or for loss of data, profits or goodwill. We also aren't liable for harm arising from your interactions with other users. Our total liability for any claim relating to the Service is limited to the greater of the amount you paid us in the 12 months before the claim or US$100.</p>
<p>Some places don't allow these exclusions or limits. Nothing in these terms limits liability that can't legally be limited, or removes rights you have under the mandatory consumer laws of the place you live.</p>

<h2>13. Changes to these terms</h2>
<p>We may update these terms. We'll change the date at the top, and if the change is significant we'll tell you in the app or by email before it takes effect. If you keep using the Service after that, the updated terms apply. If you don't agree with them, you can delete your account.</p>

<h2>14. Contact and disputes</h2>
<p>If you have a problem, please write to <a href="mailto:support@heros-agent.space">support@heros-agent.space</a> first so we can try to resolve it directly. If part of these terms can't be enforced, the rest still applies. If we don't enforce a term right away, we can still enforce it later.</p>`

const privacyHTML = `<h1>Privacy Policy</h1>
<p class="meta">Last updated November 3, 2026</p>
<p>This policy explains what personal data Heros Agents ("we", "us") collects through the SynAgent app and this website, how we use it, who receives it, and the choices you have. Questions and requests go to <a href="mailto:support@heros-agent.space">support@heros-agent.space</a>.</p>

<h2>Summary</h2>
<ul>
<li>We don't sell your personal data, show ads, or use advertising or analytics trackers in the app.</li>
<li>Your messages, profile and photos are processed by an AI model provider in China so that Suki can work and so that content can be moderated.</li>
<li>Other people's feedback after dates can affect who you are introduced to. These internal evaluations aren't shown in the app.</li>
<li>You can delete your account and export much of your data in Settings.</li>
</ul>

<h2>Information we collect</h2>
<table>
<tr><th>Type</th><th>What it includes</th></tr>
<tr><td>Account</td><td>Email address, a salted hash of your password, name, language, the time you accepted our terms, and whether your email is verified.</td></tr>
<tr><td>Profile</td><td>Birthdate, gender, pronouns, who you're looking for, city, age range, relationship goal, bio, job, education, height, smoking and drinking habits, whether you have or want children, religion, political views, languages, interests, prompts and dealbreakers. You choose what to fill in beyond what the app requires.</td></tr>
<tr><td>Photos</td><td>The photos you upload. We re-encode them, which removes location and other data embedded in the file.</td></tr>
<tr><td>Conversations</td><td>Your conversations with Suki and your messages with matches.</td></tr>
<tr><td>What Suki learns</td><td>Notes she keeps about you, your accepted and declined introductions, and what you tell her after dates. We also derive a model of your preferences from these.</td></tr>
<tr><td>Date plans</td><td>Places you consider and the times and plans you confirm.</td></tr>
<tr><td>Safety records</td><td>Reports you make or that others make about you, blocks, and the results of automated content checks.</td></tr>
<tr><td>Device and technical</td><td>IP address and device or browser description for each signed-in session, and your push notification token and device platform. We also log failed sign-in attempts, including the email address used.</td></tr>
</table>
<p>We don't use your device's GPS: you type your city, and we look up its approximate coordinates.</p>
<p>Some of this data may be sensitive, for example who you're looking for, which can suggest your sexual orientation, and your religion or political views. Providing the optional items is your choice, and you can remove them from your profile at any time.</p>

<h2>How we use it</h2>
<ul>
<li><strong>To provide the Service:</strong> your account, Suki's conversations, introductions, messaging, date planning, voice playback and notifications.</li>
<li><strong>To personalize it:</strong> learning what kind of person suits you and improving future introductions.</li>
<li><strong>To keep people safe:</strong> checking content, handling reports, and preventing abuse, fraud and repeat offenses.</li>
<li><strong>To run the Service:</strong> sending account emails, fixing problems, securing our systems and keeping backups.</li>
<li><strong>To meet legal duties:</strong> responding to lawful requests and protecting our rights and those of others.</li>
</ul>
<p>We don't use your data to train our own AI models. If you live in the EEA or UK, our legal bases are performing our contract with you, our legitimate interests in running a safe service, legal obligations, and your consent for optional sensitive details you choose to give us. You can withdraw that consent by removing the details.</p>

<h2 id="matching">How matching and automated decisions work</h2>
<p>Suki introduces people automatically in the same city. Your preferences and dealbreakers filter who can be considered. The people who remain are ranked using your profile, what you've accepted and declined, what you've told her after dates, and an AI model's assessment of how well you might suit each other.</p>
<p><strong>Post-date feedback from others can influence who you're introduced to.</strong> Its effect is limited, and it needs feedback from several people. These internal evaluations aren't displayed in the app. We don't use race or ethnicity, we don't score photos for looks, and we use religion and politics only as a dealbreaker you set yourself. Age and height can be used in matching.</p>
<p>You can ask how an automated decision about you was made, or ask for a person to review it, by writing to <a href="mailto:support@heros-agent.space">support@heros-agent.space</a>.</p>

<h2>Content moderation</h2>
<p>Explicit content isn't allowed. Messages, profile text and photos are checked automatically, including by our AI model provider. Messages that fail are not delivered. Explicit photos are deleted, and photos can also be refused for other reasons, such as showing contact details or no face. Automated checks can make mistakes and can miss content. Serious violations, or statements suggesting that a user is under 18, can place an account on hold automatically and send it to a moderator. Only a person can ban an account.</p>

<h2>Conversation coaching</h2>
<p>Suki reads your messages with a match to offer coaching to you both. This is on by default, and you can turn it off with "Let Suki read my chats" in Settings. Your words are then no longer used for coaching either of you. Safety checks still apply to all messages, including automated review of recent messages in a conversation that has been flagged.</p>

<h2>Who can see your information</h2>
<ul>
<li><strong>People you're introduced to and match with</strong> see your public profile and photos, and matches see the messages you send them. Your exact birthdate, email address and dealbreakers aren't shown to them, and neither is what you tell Suki privately.</li>
<li><strong>Our moderators and administrators</strong> can review reports, the content included in them, and records of automated safety decisions, and can access other data where needed to run or secure the Service.</li>
<li><strong>Service providers</strong>, listed below, process data for us.</li>
<li><strong>Authorities and others</strong>, where the law requires it or where we need to protect someone from serious harm. We report suspected child exploitation.</li>
<li><strong>A successor</strong>, if the Service is transferred as part of a reorganization or sale. We would tell you first.</li>
</ul>

<h2>Service providers</h2>
<table>
<tr><th>Provider</th><th>Purpose</th><th>Data</th></tr>
<tr><td>Alibaba Cloud (Beijing, China)</td><td>AI model that powers Suki, matching, coaching and moderation</td><td>Messages to Suki, messages between matches, profile details and photos</td></tr>
<tr><td>Amazon Web Services (United States)</td><td>Hosting, photo storage and backups</td><td>All the data described in this policy</td></tr>
<tr><td>Fish Audio</td><td>Suki's voice, only when you play it or turn on auto-play</td><td>The text of Suki's reply. On the plan we use, Fish Audio may use it to improve its models, so you can leave the voice off in Settings.</td></tr>
<tr><td>Expo, Apple and Google</td><td>Push notifications</td><td>Device token, notification title and text, and the screen to open. Notifications don't contain message text, though a match notification can include the other person's name.</td></tr>
<tr><td>OpenStreetMap services, including Nominatim, Komoot Photon and public Overpass servers</td><td>Finding your city and places for dates</td><td>City name or coordinates and the type of place. These requests come from our server and don't include your name.</td></tr>
<tr><td>GitHub</td><td>Checking for and downloading new versions of the app</td><td>Your IP address and device details, sent by your device when it asks for the latest version. No account data</td></tr>
<tr><td>Email recipients' mail providers</td><td>Delivering account email</td><td>Your email address and the message we send you</td></tr>
</table>

<h2>International transfers</h2>
<p>We store data in the United States and send the data shown above to China for AI processing. These countries may not have the same data protection laws as where you live, and authorities there may be able to request access to data under local law. By using the Service, you understand that your data is processed in these countries. Where the law requires it, we will put appropriate safeguards in place for these transfers.</p>

<h2 id="retention">How long we keep data</h2>
<table>
<tr><th>Data</th><th>How long</th></tr>
<tr><td>Your account, profile, photos and conversations</td><td>Until you delete them or your account. We don't currently delete inactive accounts automatically.</td></tr>
<tr><td>Signed-in sessions</td><td>Up to 30 days unless renewed or revoked.</td></tr>
<tr><td>Backups</td><td>Up to 30 days after a backup is made, so deleted data can remain in backups for that long.</td></tr>
<tr><td>Reports</td><td>Kept after an account is deleted, with the link to the account removed. They can include an excerpt of the reported content and may name a person.</td></tr>
<tr><td>Banned accounts</td><td>A hash of the email address and the account's data are kept to prevent repeat offenses and handle disputes.</td></tr>
<tr><td>Other records</td><td>Failed sign-in records and feedback you send us may be kept for security and support. Notifications already sent to others may still mention you.</td></tr>
</table>
<p>Deleting your account also deletes your conversations with your matches, for both people.</p>

<h2>Your choices and rights</h2>
<p>In the app you can:</p>
<ul>
<li>edit your profile (your birthdate can't be changed once set; write to us if it's wrong);</li>
<li>read and delete Suki's notes about you, and reset what she has learned;</li>
<li>turn off coaching and post-date check-ins, and leave her voice off;</li>
<li>block and report people, and see and end your signed-in sessions;</li>
<li>export your account details, profile, photo records, settings, conversation with Suki, messages you sent and her notes. The export doesn't include image files or internal evaluations;</li>
<li><a href="/legal/delete-account">delete your account</a>.</li>
</ul>
<p>Depending on where you live, you may also have the right to access, correct, delete or receive a copy of your data, to object to or restrict certain processing, to withdraw consent, and not to be subject to certain automated decisions. To use these rights, write to <a href="mailto:support@heros-agent.space">support@heros-agent.space</a> from your account's email address. We may need to confirm that it's you, and we won't treat you differently for asking. You can also complain to your local data protection authority.</p>

<h2>Security</h2>
<p>Connections to the Service use HTTPS. Passwords are stored as Argon2id hashes, session tokens are stored as hashes, and photos are kept in a private storage bucket with encryption at rest. Access to production systems is restricted. No system is completely secure, so please use a unique password. If a breach affects your data, we'll notify you and the authorities as the law requires.</p>

<h2>Children</h2>
<p>SynAgent is only for adults 18 and over. We don't knowingly collect data from anyone younger. If you believe a minor is using the Service, write to us, and we'll hold the account and delete its data as appropriate.</p>

<h2>Changes to this policy</h2>
<p>We'll change the date at the top when we update this policy. If a change is significant, we'll tell you in the app or by email before it takes effect.</p>

<h2>Contact</h2>
<p>Heros Agents, <a href="mailto:support@heros-agent.space">support@heros-agent.space</a></p>`

const deleteHTML = `<h1>Delete your SynAgent account</h1>
<p class="meta">Applies to SynAgent on iOS and Android</p>
<h2>In the app</h2>
<p>Open <strong>Settings &gt; Account &gt; Delete account</strong> and confirm with your password. Deletion takes effect right away and can't be undone.</p>
<h2>Without the app</h2>
<p>Email <a href="mailto:support@heros-agent.space">support@heros-agent.space</a> from your account's email address and ask us to delete it. We may ask you to confirm the request.</p>
<h2>What is deleted</h2>
<p>Your profile, photos, Suki's notes, what she has learned about you, your conversation with her, and your conversations with matches, which also disappear for the other person.</p>
<h2>What may be kept</h2>
<ul>
<li>Backups, for up to 30 days.</li>
<li>Reports involving you, without the link to your account. They can include an excerpt of the reported content.</li>
<li>Failed sign-in records and feedback you sent us, for security and support.</li>
</ul>
<p>See the <a href="/legal/privacy#retention">Privacy Policy</a> for details.</p>`
