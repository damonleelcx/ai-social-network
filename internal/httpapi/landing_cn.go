package httpapi

import (
	"encoding/json"
	"regexp"
	"strings"
)

// The China copy (suki.zhiyuwujie.top) presents SynAgent as what it is there: finding
// company for things to do, used through the WeChat mini program or the browser. The
// home page is rewritten for it when it is served: no app-store or file downloads, no
// romance, no services outside China, and only Chinese and English.

type cnChange struct{ en, newEn, zh string }

// CNBanned is what the China site must never say: romance, services or storage outside
// China, the US contact address, or app downloads outside China's app channels.
var CNBanned = regexp.MustCompile(`(?i)\bdat(e|es|ed|ing)\b|matchmak|\bmatch(es)?\b|swipe|winggirl|heros-agent|google|app store|github|\.apk|\.ipa|altstore|sideload|\baws\b|amazon|united states|\bexpo\b|stripe|fish audio|约会|红娘|相亲|交友|恋爱|僚机|美国|境外|海外|데이트|デート|매치메이커|マッチメイカー|仲人`)

// cnText are the home page's sentences that change in China, by their English. The
// Chinese is what the page shows there.
var cnText = []cnChange{
	{"SynAgent: a matchmaker you talk to, not a deck you swipe", "SynAgent: find company for the things you want to do", "SynAgent：找到一起做事的搭子"},
	{"SynAgent is built around Suki, an AI matchmaker. She introduces you to people in your city to date, or finds you company for a movie, a trip or a business contact. No swiping.",
		"SynAgent is built around Suki, an AI assistant for finding company. Tell her what you'd like to do, and she finds people in your city to do it with.",
		"SynAgent 以 AI 助手 Suki 为核心。告诉她你想做什么，她会帮你找到同城的搭子。"},
	{"A matchmaker you talk to,", "An assistant you talk to,", "一位能聊天的找搭子助手，"},
	{"not a deck you swipe.", "not a list you scroll.", "不是一串刷不完的列表。"},
	{"No swiping. No endless decks.", "No endless scrolling.", "不用没完没了地刷。"},
	{"Just someone worth meeting.", "Just good company.", "只有合拍的搭子。"},
	{"Tell Suki about you. She introduces you to people in your city to date, or finds you company for a movie, a trip or a business contact.",
		"Tell Suki what you'd like to do. She finds you company in your city for a movie, a meal, a trip, a workout or a study session.",
		"告诉 Suki 你想做什么。她会帮你在同城找到一起看电影、吃饭、旅行、运动或学习的搭子。"},
	{"Download the app", "Open in WeChat", "在微信中打开"},
	{"In your browser, or by direct download.", "In your browser, or in WeChat.", "在浏览器中，或在微信里。"},
	{"Playful · Curious · Your AI winggirl", "Playful · Curious · Your AI company-finder", "爱玩 · 好奇 · 你的找搭子助手"},
	{`"Tell me about you, and I'll introduce you to people worth meeting."`, `"Tell me what you'd like to do, and I'll find you good company."`, "“告诉我你想做什么，我帮你找合拍的搭子。”"},
	{"* Say yes or pass. It's only a match when you both say yes.", "* Say yes or pass. You team up only when you both say yes.", "* 说好或跳过。只有你们都说好，才算结伴成功。"},
	{"One matchmaker.", "One assistant.", "一位找搭子助手，"},
	{"Swipes: 0", "Forms: 0", "表单：0"},
	{"Your type, in her words", "What you enjoy, in her words", "你的喜好，用她的话说"},
	{"Wants someone curious", "Likes curious company", "喜欢好奇心强的搭子"},
	{"Most apps hand you a deck. SynAgent gives you someone to talk to, who gets a better sense of your type each time.",
		"Most apps hand you a list. SynAgent gives you someone to talk to, who learns what you enjoy each time.",
		"大多数应用只给你一串列表。SynAgent 给你一个可以聊天的助手，每次都更懂你喜欢什么。"},
	{"She learns your type", "She learns what you enjoy", "她会了解你的喜好"},
	{"Her introductions improve with what you accept, what you pass on, and what you say after your dates.",
		"Her introductions improve with what you accept, what you pass on, and what you say after you meet up.",
		"你接受了谁、跳过了谁、见面后说了什么，都会让她的介绍越来越好。"},
	{"Not only dates", "Company for anything", "不止一种搭子"},
	{"She shortlists public places you'd both enjoy. You each pick a favorite privately, and the date is set once you both confirm.",
		"She shortlists public places you'd both enjoy. You each pick a favorite privately, and the plan is set once you both confirm.",
		"她会筛选你们都会喜欢的公共场所。你们各自私下选出最喜欢的地方，双方确认后见面就定下来了。"},
	{"Date", "Plan", "计划"},
	{"Download", "Get it", "使用方式"},
	{"IV · Download", "IV · Get it", "IV · 使用方式"},
	// The home page's script.
	{"* It's only a match when you both say yes.", "* You team up only when you both say yes.", "* 只有你们都说好，才算结伴成功。"},
	{"She looks for people in your city who suit you, and explains why she thinks you'd get along. It's only a match when you both say yes.",
		"She looks for people in your city who suit you, and explains why she thinks you'd get along. You team up only when you both say yes.",
		"她会在你所在的城市寻找适合你的人，并说明为什么觉得你们合得来。只有你们都说好，才算结伴成功。"},
	{"It's only a match when you both say yes.", "You team up only when you both say yes.", "只有你们都说好，才算结伴成功。"},
	{"Chat with your match directly. She can suggest an opener or a question when things go quiet, and you can turn that off.",
		"Chat with them directly. She can suggest an opener or a question when things go quiet, and you can turn that off.",
		"直接和搭子聊天。冷场时她可以建议开场白或问题，你也可以关闭这项功能。"},
	{"Plan the date", "Plan the meetup", "安排见面"},
	{"She shortlists public places based on what you both enjoy. You each pick a favorite privately, and the date is set once you both confirm.",
		"She shortlists public places based on what you both enjoy. You each pick a favorite privately, and the plan is set once you both confirm.",
		"她会根据你们共同的喜好筛选公共场所。你们各自私下选出最喜欢的地方，双方确认后见面就定下来了。"},
	{"What you share after a date helps her make better introductions next time.",
		"What you share after meeting up helps her make better introductions next time.",
		"见面后你分享的内容，能帮她下次做出更好的介绍。"},
	{"Take her with you", "Take her with you", "随时找她"},
	{"Your move", "Your move", "轮到你了"},
	{"Write to us", "Write to us", "联系我们"},
}

// cnDownload replaces the download section: WeChat and the browser.
const cnDownload = `<section class="solid" id="download" aria-labelledby="download-title">
    <div class="wrap">
      <div class="divider"><i data-sprite="heart" class="spr flick"></i><span>IV · Get it</span><i data-sprite="heart" class="spr flick"></i></div>
      <h2 id="download-title" class="big">Take her with you</h2>
      <p class="section-lede">Use SynAgent in WeChat or in your browser. For adults 18 and over. Your first introduction is free. After that, messages to Suki use credits, and chatting with the people you team up with stays free.</p>
      <div class="downloads">
        <article class="download download-web">
          <h3>In WeChat</h3>
          <p>Search for HelloSuki in WeChat, or open the mini program from a friend's share.</p>
        </article>
        <article class="download">
          <h3>In your browser</h3>
          <p>Nothing to install, and it works on a computer too. Your account is the same one the mini program uses.</p>
          <a class="btn btn-heart" href="/app/">Open SynAgent <i data-sprite="heart" class="spr"></i></a>
        </article>
      </div>
    </div>
  </section>`

var cnDownloadText = []cnChange{
	{"Use SynAgent in WeChat or in your browser. For adults 18 and over. Your first introduction is free. After that, messages to Suki use credits, and chatting with the people you team up with stays free.", "",
		"在微信或浏览器中使用 SynAgent。仅限 18 岁及以上。第一次介绍免费，之后和 Suki 对话消耗积分，和搭子聊天一直免费。"},
	{"In WeChat", "", "在微信里"},
	{"Search for HelloSuki in WeChat, or open the mini program from a friend's share.", "", "在微信中搜索 HelloSuki，或从朋友分享的小程序打开。"},
	{"Nothing to install, and it works on a computer too. Your account is the same one the mini program uses.", "", "无需安装，电脑上也能用。账号和小程序是同一个。"},
}

var cnDownloadSection = regexp.MustCompile(`(?s)<section class="solid" id="download".*?</section>`)

// chinaLanding rewrites one of the home page's files for the China copy.
func chinaLanding(name string, body []byte) []byte {
	s := string(body)
	switch name {
	case "landing/index.html":
		s = cnDownloadSection.ReplaceAllLiteralString(s, cnDownload)
		s = strings.ReplaceAll(s, `      <option value="ko">한국어</option>
`, "")
		s = strings.ReplaceAll(s, `      <option value="ja">日本語</option>
`, "")
		s = strings.ReplaceAll(s, `if (lang === 'zh' || lang === 'ko' || lang === 'ja') {`, `if (lang === 'zh') {`)
		s = strings.ReplaceAll(s, `if (base === 'zh' || base === 'ko' || base === 'ja' || base === 'en') lang = base;`, `if (base === 'zh' || base === 'en') lang = base;`)
		s = strings.ReplaceAll(s, `<a class="btn btn-ghost" href="#download">Download the app</a>`, `<a class="btn btn-ghost" href="#download">Open in WeChat</a>`)
		s = strings.ReplaceAll(s, `<a class="btn btn-heart" href="#download">Download <i`, `<a class="btn btn-heart" href="/app/">Open SynAgent <i`)
		s = strings.ReplaceAll(s, "mailto:support@heros-agent.space", "mailto:support@zhiyuwujie.top")
		s = replaceText(s, false)
	case "landing/app.js":
		s = replaceText(s, true)
	case "landing/i18n.js":
		s = chinaTranslations(s)
	}
	return []byte(s)
}

// replaceText swaps whole sentences, longest first so a short one never cuts into a long one.
func replaceText(s string, script bool) string {
	all := append(append([]cnChange{}, cnText...), cnDownloadText...)
	for i := range all {
		for j := i + 1; j < len(all); j++ {
			if len(all[j].en) > len(all[i].en) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	for _, c := range all {
		if c.newEn == "" || c.newEn == c.en {
			continue
		}
		if script {
			s = strings.ReplaceAll(s, strings.ReplaceAll(c.en, "'", `\'`), strings.ReplaceAll(c.newEn, "'", `\'`))
			s = strings.ReplaceAll(s, "'"+c.en+"'", "'"+c.newEn+"'")
			s = strings.ReplaceAll(s, `"`+c.en+`"`, `"`+c.newEn+`"`)
			continue
		}
		// Only as a whole text node or attribute value, so "Date" never touches "Update".
		for _, wrap := range [][2]string{{">", "<"}, {`"`, `"`}, {"<title>", "</title>"}} {
			s = strings.ReplaceAll(s, wrap[0]+c.en+wrap[1], wrap[0]+c.newEn+wrap[1])
		}
		s = strings.ReplaceAll(s, "<br>"+c.en+"<", "<br>"+c.newEn+"<")
		s = strings.ReplaceAll(s, ">"+c.en+"<br>", ">"+c.newEn+"<br>")
		s = strings.ReplaceAll(s, "</i>"+c.en+"<br>", "</i>"+c.newEn+"<br>")
	}
	return s
}

// A row of the translation table: key: [zh, ko, ja],
var i18nRow = regexp.MustCompile(`^(\s*)('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"|[A-Za-z_][A-Za-z0-9_]*)\s*:\s*\[\s*('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*")\s*,.*\],?\s*$`)

func jsKey(raw string) string {
	if strings.HasPrefix(raw, "'") || strings.HasPrefix(raw, `"`) {
		return strings.ReplaceAll(strings.ReplaceAll(raw[1:len(raw)-1], `\'`, "'"), `\"`, `"`)
	}
	return raw
}

func jsString(s string) string { b, _ := json.Marshal(s); return string(b) }

// chinaTranslations keeps only the Chinese column, with the China wording, and renames the
// rows whose English changed.
func chinaTranslations(s string) string {
	byEn := map[string]cnChange{}
	for _, c := range append(append([]cnChange{}, cnText...), cnDownloadText...) {
		byEn[c.en] = c
		if c.newEn != "" {
			byEn[c.newEn] = cnChange{en: c.newEn, newEn: c.newEn, zh: c.zh}
		}
	}
	lines := strings.Split(s, "\n")
	seen := map[string]bool{}
	out := lines[:0]
	for _, line := range lines {
		m := i18nRow.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			out = append(out, line)
			continue
		}
		key, zh := jsKey(m[2]), m[3]
		if c, ok := byEn[key]; ok {
			if c.newEn != "" {
				key = c.newEn
			}
			zh = jsString(c.zh)
		} else if CNBanned.MatchString(key) || CNBanned.MatchString(zh) {
			continue // only for parts of the page China doesn't show
		}
		seen[key] = true
		out = append(out, m[1]+jsString(key)+": ["+zh+`, "", ""],`)
	}
	// Sentences that only exist in China.
	var extra []string
	for _, c := range cnDownloadText {
		if !seen[c.en] {
			extra = append(extra, "  "+jsString(c.en)+": ["+jsString(c.zh)+`, "", ""],`)
		}
	}
	for _, c := range cnText {
		if c.newEn != "" && !seen[c.newEn] {
			extra = append(extra, "  "+jsString(c.newEn)+": ["+jsString(c.zh)+`, "", ""],`)
			seen[c.newEn] = true
		}
	}
	joined := strings.Join(out, "\n")
	if len(extra) > 0 {
		joined = strings.Replace(joined, "\n};", "\n"+strings.Join(extra, "\n")+"\n};", 1)
	}
	return joined
}
