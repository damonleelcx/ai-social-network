// Package mail sends through the cluster's relay over STARTTLS, or logs when
// the relay is not configured so development never silently drops a message.
package mail

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"html"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"strings"

	"github.com/damonleelcx/ai-social-network/internal/i18n"
	"time"
)

type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
	ReplyTo string
	// MessageID is set by the caller when the send is idempotent: the same
	// logical message always carries the same id, so a receiving system that
	// deduplicates on it sees one message.
	MessageID string
}

type Mailer interface {
	Send(ctx context.Context, m Message) (messageID string, err error)
	Enabled() bool
}

type SMTP struct {
	Host, User, Pass, From, ReplyTo string
	Port                            int
}

func (s *SMTP) Enabled() bool { return true }

func (s *SMTP) Send(ctx context.Context, m Message) (string, error) {
	if m.MessageID == "" {
		b := make([]byte, 12)
		_, _ = rand.Read(b)
		m.MessageID = hex.EncodeToString(b)
	}
	domain := s.From[strings.LastIndexByte(s.From, '@')+1:]
	msgID := fmt.Sprintf("<%s@%s>", m.MessageID, domain)
	if m.ReplyTo == "" {
		m.ReplyTo = s.ReplyTo
	}
	raw := build(s.From, m, msgID)

	d := net.Dialer{Timeout: 15 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", s.Host, s.Port))
	if err != nil {
		return "", err
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	} else {
		_ = conn.SetDeadline(time.Now().Add(45 * time.Second))
	}
	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		conn.Close()
		return "", err
	}
	defer c.Close()
	// The relay's certificate is for its public name, which is what Host is;
	// verification is on. A relay that cannot prove its name gets no password.
	if err := c.StartTLS(&tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}); err != nil {
		return "", fmt.Errorf("starttls: %w", err)
	}
	if err := c.Auth(smtp.PlainAuth("", s.User, s.Pass, s.Host)); err != nil {
		return "", fmt.Errorf("auth: %w", err)
	}
	if err := c.Mail(s.From); err != nil {
		return "", err
	}
	if err := c.Rcpt(m.To); err != nil {
		return "", err
	}
	w, err := c.Data()
	if err != nil {
		return "", err
	}
	if _, err := w.Write(raw); err != nil {
		return "", err
	}
	// Close returns the relay's reply to DATA: a 250 here is the relay taking
	// responsibility for the message, which is what "sent" means.
	if err := w.Close(); err != nil {
		return "", err
	}
	_ = c.Quit()
	return msgID, nil
}

func build(from string, m Message, msgID string) []byte {
	var b strings.Builder
	boundary := "play-" + strings.Trim(msgID, "<>")
	boundary = strings.NewReplacer("@", "-", ".", "-").Replace(boundary)
	h := func(k, v string) { b.WriteString(k + ": " + v + "\r\n") }
	h("From", mime.QEncoding.Encode("utf-8", SenderName)+" <"+from+">")
	h("To", m.To)
	if m.ReplyTo != "" {
		h("Reply-To", m.ReplyTo)
	}
	h("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	h("Message-ID", msgID)
	h("Date", time.Now().UTC().Format(time.RFC1123Z))
	h("MIME-Version", "1.0")
	if m.HTML == "" {
		h("Content-Type", "text/plain; charset=utf-8")
		h("Content-Transfer-Encoding", "8bit")
		b.WriteString("\r\n" + crlf(m.Text))
		return []byte(b.String())
	}
	h("Content-Type", `multipart/alternative; boundary="`+boundary+`"`)
	b.WriteString("\r\n")
	for _, part := range []struct{ ct, body string }{{"text/plain", m.Text}, {"text/html", m.HTML}} {
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: " + part.ct + "; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n")
		b.WriteString(crlf(part.body) + "\r\n")
	}
	b.WriteString("--" + boundary + "--\r\n")
	return []byte(b.String())
}

func crlf(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// A lone "." on a line would end the DATA section early.
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, ".") {
			lines[i] = "." + l
		}
	}
	return strings.Join(lines, "\r\n")
}

// Check proves the relay path end to end — TCP, STARTTLS with certificate
// verification, AUTH — and quits without sending anything.
func (s *SMTP) Check(ctx context.Context) error {
	d := net.Dialer{Timeout: 15 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", s.Host, s.Port))
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	c, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("greeting: %w", err)
	}
	defer c.Close()
	if err := c.StartTLS(&tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}
	if err := c.Auth(smtp.PlainAuth("", s.User, s.Pass, s.Host)); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	return c.Quit()
}

// Log is the development mailer: it prints the message, links included, so a
// developer can click through verification without a relay.
type Log struct{}

func (Log) Enabled() bool { return false }

func (Log) Send(_ context.Context, m Message) (string, error) {
	slog.Info("MAIL (not sent — relay not configured)", "to", m.To, "subject", m.Subject, "body", m.Text)
	return "log-" + m.MessageID, nil
}

// SenderName is the display name on every message: Suki writes the emails.
const SenderName = "Suki · SynAgent"

// Signature is how Suki signs off. Her name is the same in every language.
func Signature(string) string { return "— Suki · SynAgent" }

// footerNote is the footer line under every email, per language.
var footerNote = map[string]string{
	"en": "You're receiving this because you have a SynAgent account.",
	"zh": "你收到这封邮件，是因为你有一个 SynAgent 账户。",
	"ko": "SynAgent 계정이 있어서 이 메일을 보내 드렸어요.",
	"ja": "SynAgent のアカウントをお持ちのため、このメールをお送りしています。",
}

// htmlLang is the lang attribute of the email page.
func htmlLang(lang string) string {
	if _, ok := footerNote[lang]; ok {
		return lang
	}
	return "en"
}

// Page is the branded HTML frame every email uses: the wordmark, a title, a
// lead paragraph, one call-to-action button, the link spelled out, a footnote
// and Suki's sign-off. Navy and blue to match the site. Table layout and
// inline styles because that is what mail clients render reliably.
func Page(lang, title, lead, cta, link, foot string) string {
	e := html.EscapeString
	return `<!doctype html><html lang="` + htmlLang(lang) + `"><body style="margin:0;background:#0b1430;font-family:-apple-system,Segoe UI,Helvetica,Arial,sans-serif;color:#e8eeff">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#0b1430"><tr><td align="center" style="padding:40px 16px">
<table role="presentation" width="520" cellpadding="0" cellspacing="0" style="max-width:520px;background:#121e45;border:1px solid #23346e;border-radius:20px;padding:36px">
<tr><td style="font-size:12px;letter-spacing:.22em;font-weight:700;color:#4f8cff">SYNAGENT</td></tr>
<tr><td style="padding-top:18px;font-size:26px;font-weight:700;line-height:1.25;color:#ffffff">` + e(title) + `</td></tr>
<tr><td style="padding-top:14px;font-size:15px;line-height:1.65;color:#c3cdee">` + e(lead) + `</td></tr>
<tr><td style="padding-top:26px"><a href="` + e(link) + `" style="display:inline-block;background:#4f8cff;color:#ffffff;text-decoration:none;padding:14px 28px;border-radius:999px;font-weight:600">` + e(cta) + `</a></td></tr>
<tr><td style="padding-top:22px;font-size:12px;line-height:1.6;color:#7f8bb5;word-break:break-all">` + e(link) + `</td></tr>
<tr><td style="padding-top:18px;font-size:13px;line-height:1.6;color:#7f8bb5">` + e(foot) + `</td></tr>
<tr><td style="padding-top:26px;font-size:14px;color:#3fd2ff">` + e(Signature(lang)) + `</td></tr>
</table>
<p style="margin:18px 0 0;font-size:11px;color:#56618a">` + e(footerNote[htmlLang(lang)]) + `</p>
</td></tr></table></body></html>`
}

// CodePage is the HTML body of an email whose whole point is one code.
func CodePage(lang, title, lead, code, foot string) string {
	return fmt.Sprintf(`<!doctype html><html lang="%s"><body style="font-family:system-ui,sans-serif;background:#f4f1fb;padding:24px;color:#2a2438">
<div style="max-width:480px;margin:auto;background:#fff;border-radius:16px;padding:28px">
<h1 style="font-size:20px;margin:0 0 12px">%s</h1>
<p style="line-height:1.5">%s</p>
<p style="font-size:34px;letter-spacing:8px;font-weight:700;text-align:center;margin:24px 0">%s</p>
<p style="color:#6b6480;font-size:13px">%s %s</p>
<p style="color:#6b6480;font-size:13px">Suki</p>
</div></body></html>`, htmlLang(lang), html.EscapeString(title), html.EscapeString(lead), html.EscapeString(code), html.EscapeString(i18n.T(lang, "It works for 15 minutes.")), html.EscapeString(foot))
}
