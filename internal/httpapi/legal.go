package httpapi

import (
	"embed"
	"net/http"
	"regexp"
	"strings"
)

// The same theme choice as the landing page: their pick if they made one, else the system's.
const themeScript = `<script>(function(){var s=null;try{s=localStorage.getItem('synagent.theme')}catch(e){}document.documentElement.dataset.theme=s==='light'||s==='dark'?s:matchMedia('(prefers-color-scheme: light)').matches?'light':'dark'})()</script>`

const pageStyle = `:root,[data-theme=light]{color-scheme:light;--ink:#2a2438;--bg:#fff;--link:#5b3fb5;--soft:#6b6478;--line:#e6e2ee}[data-theme=dark]{color-scheme:dark;--ink:#f1ecfb;--bg:#0c0720;--link:#bfa2ff;--soft:#b3a8cc;--line:#352a55}body{font-family:system-ui,-apple-system,"Segoe UI",sans-serif;max-width:720px;margin:0 auto;padding:40px 20px 56px;line-height:1.6;color:var(--ink);background:var(--bg)}h1{font-size:30px;line-height:1.2;margin:0 0 8px}h2{font-size:19px;margin:34px 0 8px}h3{font-size:16px;margin:20px 0 4px}ul{padding-left:22px}li{margin:6px 0}a{color:var(--link)}.meta{color:var(--soft);font-size:14px;margin:0 0 24px}table{border-collapse:collapse;width:100%;font-size:15px}th,td{text-align:left;vertical-align:top;border-bottom:1px solid var(--line);padding:8px 10px 8px 0}footer{margin-top:44px;padding-top:16px;border-top:1px solid var(--line);font-size:14px;color:var(--soft)}footer a{margin-right:14px}`

// The pages live in legal/<language>/, English being the original.
//
//go:embed legal legal-cn
var legalFiles embed.FS

// The China copy has its own pages (legal-cn/), written for it: Chinese, the binding text,
// and English.
var chinaLegalLanguages = []legalChrome{
	{code: "zh", name: "中文", terms: "用户协议", privacy: "隐私政策", deleteAccount: "注销账户", contact: "联系我们"},
	{code: "en", name: "English", terms: "Terms", privacy: "Privacy", deleteAccount: "Delete your account", contact: "Contact",
		note: "This translation is for convenience. If it differs from the Chinese version, the Chinese version prevails."},
}

// legalChrome is the words around a page, for each language.
type legalChrome struct {
	code, name, terms, privacy, deleteAccount, contact, note string
}

var legalLanguages = []legalChrome{
	{code: "en", name: "English", terms: "Terms", privacy: "Privacy", deleteAccount: "Delete your account", contact: "Contact"},
	{code: "zh", name: "中文", terms: "条款", privacy: "隐私", deleteAccount: "删除账户", contact: "联系我们",
		note: "本译文仅为方便阅读。如与英文版不一致，以英文版为准。"},
	{code: "ko", name: "한국어", terms: "약관", privacy: "개인정보", deleteAccount: "계정 삭제", contact: "문의하기",
		note: "이 번역본은 편의를 위해 제공됩니다. 영문본과 내용이 다를 경우 영문본이 우선합니다."},
	{code: "ja", name: "日本語", terms: "規約", privacy: "プライバシー", deleteAccount: "アカウントの削除", contact: "お問い合わせ",
		note: "この翻訳は参考用です。英語版と内容が異なる場合は、英語版が優先します。"},
	{code: "es", name: "Español", terms: "Términos", privacy: "Privacidad", deleteAccount: "Eliminar tu cuenta", contact: "Contacto",
		note: "Esta traducción se ofrece para tu comodidad. Si difiere de la versión en inglés, prevalece la versión en inglés."},
}

// pickLegalLanguage takes ?lang= first, then what the browser asks for, then the first.
func pickLegalLanguage(r *http.Request, languages []legalChrome) legalChrome {
	find := func(code string) (legalChrome, bool) {
		code, _, _ = strings.Cut(strings.ToLower(strings.TrimSpace(code)), "-")
		for _, l := range languages {
			if l.code == code {
				return l, true
			}
		}
		return legalChrome{}, false
	}
	if l, ok := find(r.URL.Query().Get("lang")); ok {
		return l
	}
	for _, part := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		tag, _, _ := strings.Cut(part, ";")
		if l, ok := find(tag); ok {
			return l
		}
	}
	return languages[0]
}

var legalLink = regexp.MustCompile(`href="/legal/[a-z-]+`)

// legalPage serves a page in the language picked. Its links stay in that language.
func legalPage(page string, china bool) http.HandlerFunc {
	languages, dir := legalLanguages, "legal/"
	if china {
		languages, dir = chinaLegalLanguages, "legal-cn/"
	}
	return func(w http.ResponseWriter, r *http.Request) {
		l := pickLegalLanguage(r, languages)
		body, err := legalFiles.ReadFile(dir + l.code + "/" + page + ".html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		text := string(body)
		title := "SynAgent"
		if _, rest, ok := strings.Cut(text, "<h1>"); ok {
			title, _, _ = strings.Cut(rest, "</h1>")
		}
		main := text + `<footer><a href="/">SynAgent</a><a href="/legal/terms">` + l.terms + `</a><a href="/legal/privacy">` + l.privacy + `</a><a href="/legal/delete-account">` + l.deleteAccount + `</a><a href="/legal/contact">` + l.contact + `</a>`
		main = legalLink.ReplaceAllString(main, `$0?lang=`+l.code)
		var out strings.Builder
		out.WriteString(`<!doctype html><html lang="` + l.code + `"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + title + `</title>
` + themeScript + `<style>` + pageStyle + `</style><body>`)
		if l.note != "" {
			out.WriteString(`<p class="meta">` + l.note + `</p>`)
		}
		out.WriteString(main)
		out.WriteString(`<nav aria-label="Language">`)
		for _, other := range languages {
			out.WriteString(`<a href="/legal/` + page + `?lang=` + other.code + `" lang="` + other.code + `">` + other.name + `</a>`)
		}
		out.WriteString(`</nav></footer></body></html>`)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Vary", "Accept-Language")
		_, _ = w.Write([]byte(out.String()))
	}
}
