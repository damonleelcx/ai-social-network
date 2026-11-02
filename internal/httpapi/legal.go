package httpapi

import "net/http"

func legalPage(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>SynAgent</title>
<style>body{font-family:system-ui,sans-serif;max-width:680px;margin:32px auto;padding:0 16px;line-height:1.55;color:#2a2438}h1{font-size:24px}h2{font-size:18px;margin-top:28px}</style><body>` + body + `</body></html>`))
	}
}

const openHTML = `<h1>SynAgent</h1><p>Open the SynAgent app on your phone to continue.</p>`

const termsHTML = `<h1>SynAgent Terms of Use</h1>
<p><em>Draft. Have a lawyer review this before a public launch.</em></p>
<h2>Who can use SynAgent</h2><p>You must be at least 18. One account per person, and everything in your profile must be true and about you.</p>
<h2>No tolerance for objectionable content or abusive users</h2><p>Harassment, threats, hate, nudity or sexual content, scams, requests for money, impersonation and anything involving minors are not allowed. Every message and profile can be reported, and anyone can be blocked. Our moderators act on reports within 24 hours and can remove content and accounts.</p>
<h2>Suki</h2><p>Suki is an AI. She can be wrong. Her suggestions aren't professional advice and she doesn't guarantee who you will meet.</p>
<h2>Meeting people</h2><p>You are responsible for your own safety when you meet someone. We don't run criminal background checks. Meet in public, tell a friend, and leave whenever you want.</p>
<h2>Ending your account</h2><p>You can delete your account at any time in Settings. We can suspend accounts that break these terms.</p>
<h2>Contact</h2><p>support@heros-agent.space</p>`

const privacyHTML = `<h1>SynAgent Privacy Policy</h1>
<p><em>Draft. Have a lawyer review this before a public launch.</em></p>
<h2>What we collect</h2><p>Your email and password (stored as a hash); what you put in your profile; your photos; your conversations with Suki and with your matches; and what you tell Suki about your dates.</p>
<h2>Where it is stored</h2><p>Account and profile data in our PostgreSQL database, chat messages in our MongoDB database, and photos in a private Amazon S3 bucket. Photos are re-encoded on upload, which removes location data embedded in the file.</p>
<h2>AI processing</h2><p>Your messages to Suki, your profile, and your photos (for moderation) are sent to our AI model provider to produce answers and decisions. Unless you turn off "Let Suki read my chats" in Settings, your messages with a match are also used to coach you and that match. If you turn it off, your words are not used for either of you.</p>
<h2>How matching and evaluation work</h2><p>Suki introduces people automatically using your stated preferences, your profile, what you accept and decline, and what you and others tell her after dates. <strong>Post-date feedback from others can influence who you are introduced to.</strong> These internal evaluations are not displayed in the app. We do not use race or ethnicity, we do not score photos for looks, and we treat religion and politics only as a dealbreaker you set yourself. To ask about an automated decision or request human review, write to support@heros-agent.space.</p>
<h2>Content filtering</h2><p>Explicit content isn't allowed. Your messages, profile text and photos are checked automatically, including by our AI model provider, before other people can see them. Filtered messages aren't delivered and filtered photos are deleted. Our moderators may review what was blocked.</p>
<h2>Who sees what</h2><p>Others see your public profile only if Suki introduces you. Your exact birthdate, email, dealbreakers, and everything you tell Suki are not shown to anyone. Moderators can read reported conversations. A report, including an excerpt of a flagged message, may be kept after an account is deleted so we can act on abuse.</p>
<h2>Other services</h2><p>When you play Suki's voice, the text of that message is sent to Fish Audio to produce the speech. On the plan we use, Fish Audio may use that text to improve its models, so you can leave the voice off in Settings. Nothing is sent unless you play it or turn on auto-play. City and place searches, which contain your city and general interest categories but not your name, go to public OpenStreetMap services. Push notifications go through Expo, Apple and Google, and never contain message text.</p>
<h2>Your choices</h2><p>In Settings you can read and delete Suki's notes about you, reset what she has learned, turn off coaching and debriefs, export your data, and delete your account. Deleting your account removes your profile, photos and conversations, including your side of conversations with others.</p>
<h2>Contact</h2><p>support@heros-agent.space</p>`

const deleteHTML = `<h1>Delete your SynAgent account</h1>
<p>In the app, open <strong>Settings &gt; Account &gt; Delete account</strong> and confirm with your password. This permanently removes your profile, photos and conversations.</p>
<p>If you no longer have the app, email support@heros-agent.space from your account's address and we will delete it for you.</p>`
