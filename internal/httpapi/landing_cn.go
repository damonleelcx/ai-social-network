package httpapi

import (
	"encoding/json"
	"regexp"
	"strings"
)

// The China copy (suki.zhiyuwujie.top) presents SynAgent as what it is there: finding
// company for things to do, used through the WeChat mini program or the browser. The
// home page is rewritten for it when it is served: no app-store or file downloads, no
// romance and no services outside China, in every language the page offers.

// cnChange is one sentence that changes in China, by its English, with the new English
// (empty for sentences that only exist in China) and its Chinese, Korean and Japanese.
type cnChange struct{ en, newEn, zh, ko, ja string }

// CNBanned is what the China site must never say, in any of its languages: romance,
// services or storage outside China, the US contact address, or app downloads outside
// China's app channels.
var CNBanned = regexp.MustCompile(`(?i)\bdat(e|es|ed|ing)\b|matchmak|\bmatch(es)?\b|swipe|winggirl|heros-agent|google|app store|github|\.apk|\.ipa|altstore|sideload|\baws\b|amazon|united states|\bexpo\b|stripe|fish audio|约会|红娘|相亲|交友|恋爱|僚机|美国|境外|海外|데이트|매치|매칭|연애|소개팅|윙걸|미국|デート|マッチ|恋愛|婚活|出会い系|ウィングガール|アメリカ|仲人`)

var cnText = []cnChange{
	{"SynAgent: a matchmaker you talk to, not a deck you swipe", "SynAgent: find company for the things you want to do",
		"SynAgent：找到一起做事的搭子", "SynAgent: 하고 싶은 일을 함께할 친구 찾기", "SynAgent：やりたいことを一緒に楽しむ仲間探し"},
	{"SynAgent is built around Suki, an AI matchmaker. She introduces you to people in your city to date, or finds you company for a movie, a trip or a business contact. No swiping.",
		"SynAgent is built around Suki, an AI assistant for finding company. Tell her what you'd like to do, and she finds people in your city to do it with.",
		"SynAgent 以 AI 助手 Suki 为核心。告诉她你想做什么，她会帮你找到同城的搭子。",
		"SynAgent는 함께할 친구를 찾아 주는 AI 도우미 Suki를 중심으로 만들어졌어요. 하고 싶은 일을 말하면 같은 도시에서 함께할 사람을 찾아 줘요.",
		"SynAgent は、仲間探しを手伝う AI アシスタント Suki が中心のサービスです。やりたいことを伝えると、同じ街で一緒に楽しめる人を見つけてくれます。"},
	{"SynAgent is a dating app built around Suki, an AI matchmaker. Tell her about you and she introduces you to people in your city. No swiping.",
		"SynAgent is built around Suki, an AI assistant for finding company. Tell her what you'd like to do, and she finds people in your city to do it with.",
		"SynAgent 以 AI 助手 Suki 为核心。告诉她你想做什么，她会帮你找到同城的搭子。",
		"SynAgent는 함께할 친구를 찾아 주는 AI 도우미 Suki를 중심으로 만들어졌어요. 하고 싶은 일을 말하면 같은 도시에서 함께할 사람을 찾아 줘요.",
		"SynAgent は、仲間探しを手伝う AI アシスタント Suki が中心のサービスです。やりたいことを伝えると、同じ街で一緒に楽しめる人を見つけてくれます。"},
	{"A matchmaker you talk to,", "An assistant you talk to,", "一位能聊天的找搭子助手，", "끝없는 목록이 아니라,", "スクロールし続けるリストではなく、"},
	{"not a deck you swipe.", "not a list you scroll.", "不是一串刷不完的列表。", "대화하는 도우미.", "話せるアシスタント。"},
	{"No swiping. No endless decks.", "No endless scrolling.", "不用没完没了地刷。", "끝없이 넘길 필요 없어요.", "延々とスクロールしなくていい。"},
	{"Just someone worth meeting.", "Just good company.", "只有合拍的搭子。", "잘 맞는 친구만 있으면 돼요.", "気の合う仲間がいればいい。"},
	{"Tell Suki about you. She introduces you to people in your city to date, or finds you company for a movie, a trip or a business contact.",
		"Tell Suki what you'd like to do. She finds you company in your city for a movie, a meal, a trip, a workout or a study session.",
		"告诉 Suki 你想做什么。她会帮你在同城找到一起看电影、吃饭、旅行、运动或学习的搭子。",
		"Suki에게 하고 싶은 일을 말해 주세요. 같은 도시에서 영화, 식사, 여행, 운동, 공부를 함께할 친구를 찾아 줘요.",
		"Suki にやりたいことを話してください。同じ街で、映画、食事、旅行、運動、勉強を一緒にする仲間を見つけてくれます。"},
	{"Download the app", "Open in WeChat", "在微信中打开", "위챗에서 열기", "WeChat で開く"},
	{"In your browser, or by direct download.", "In your browser, or in WeChat.", "在浏览器中，或在微信里。", "브라우저 또는 위챗에서.", "ブラウザか、WeChat で。"},
	{"Playful · Curious · Your AI winggirl", "Playful · Curious · Your AI company-finder",
		"爱玩 · 好奇 · 你的找搭子助手", "장난기 · 호기심 · 함께할 친구를 찾아 주는 AI", "おちゃめ · 好奇心旺盛 · 仲間探しの AI"},
	{`"Tell me about you, and I'll introduce you to people worth meeting."`, `"Tell me what you'd like to do, and I'll find you good company."`,
		"“告诉我你想做什么，我帮你找合拍的搭子。”", "“하고 싶은 일을 말해 줘. 잘 맞는 친구를 찾아 줄게.”", "「やりたいことを教えて。気の合う仲間を見つけてあげる。」"},
	{"* Say yes or pass. It's only a match when you both say yes.", "* Say yes or pass. You team up only when you both say yes.",
		"* 说好或跳过。只有你们都说好，才算结伴成功。", "* 좋아요 또는 패스를 골라요. 두 사람 모두 좋다고 해야 함께하게 돼요.", "* イエスかパスを選んで。ふたりともイエスのときだけ一緒に行けるよ。"},
	{"One matchmaker.", "One assistant.", "一位找搭子助手，", "도우미 한 명,", "アシスタントはひとり。"},
	{"Swipes: 0", "Forms: 0", "表单：0", "양식: 0", "フォーム：0"},
	{"Your type, in her words", "What you enjoy, in her words", "你的喜好，用她的话说", "그녀가 말하는 당신의 취향", "彼女の言葉で、あなたの好み"},
	{"Wants someone curious", "Likes curious company", "喜欢好奇心强的搭子", "호기심 많은 친구를 좋아함", "好奇心旺盛な仲間が好き"},
	{"Most apps hand you a deck. SynAgent gives you someone to talk to, who gets a better sense of your type each time.",
		"Most apps hand you a list. SynAgent gives you someone to talk to, who learns what you enjoy each time.",
		"大多数应用只给你一串列表。SynAgent 给你一个可以聊天的助手，每次都更懂你喜欢什么。",
		"대부분의 앱은 목록만 줘요. SynAgent는 대화할 상대를 주고, 그녀는 매번 당신의 취향을 더 잘 알게 돼요.",
		"多くのアプリはリストを渡すだけ。SynAgent には話し相手がいて、そのたびにあなたの好みをよく知っていきます。"},
	{"She learns your type", "She learns what you enjoy", "她会了解你的喜好", "당신의 취향을 배워요", "あなたの好みを覚えます"},
	{"Her introductions improve with what you accept, what you pass on, and what you say after your dates.",
		"Her introductions improve with what you accept, what you pass on, and what you say after you meet up.",
		"你接受了谁、跳过了谁、见面后说了什么，都会让她的介绍越来越好。",
		"수락한 사람, 패스한 사람, 만난 뒤 들려준 이야기로 소개가 점점 좋아져요.",
		"受け入れた人、パスした人、会った後に話したことをもとに、紹介が良くなっていきます。"},
	{"Not only dates", "Company for anything", "不止一种搭子", "무엇이든 함께할 친구", "どんなことでも一緒に"},
	{"She shortlists public places you'd both enjoy. You each pick a favorite privately, and the date is set once you both confirm.",
		"She shortlists public places you'd both enjoy. You each pick a favorite privately, and the plan is set once you both confirm.",
		"她会筛选你们都会喜欢的公共场所。你们各自私下选出最喜欢的地方，双方确认后见面就定下来了。",
		"두 사람 모두 즐길 만한 공공장소를 추려 줘요. 각자 비공개로 고르고, 둘 다 확인하면 약속이 정해져요.",
		"ふたりが楽しめそうな公共の場所を絞り込みます。それぞれ非公開で選び、ふたりが確認すると予定が決まります。"},
	{"Date", "Plan", "计划", "약속", "予定"},
	{"Download", "Get it", "使用方式", "이용 방법", "使い方"},
	{"IV · Download", "IV · Get it", "IV · 使用方式", "IV · 이용 방법", "IV · 使い方"},
	// The home page's script.
	{"* It's only a match when you both say yes.", "* You team up only when you both say yes.",
		"* 只有你们都说好，才算结伴成功。", "* 두 사람 모두 좋다고 해야 함께하게 돼요.", "* ふたりともイエスと言ったときだけ、一緒に行けるよ。"},
	{"She looks for people in your city who suit you, and explains why she thinks you'd get along. It's only a match when you both say yes.",
		"She looks for people in your city who suit you, and explains why she thinks you'd get along. You team up only when you both say yes.",
		"她会在你所在的城市寻找适合你的人，并说明为什么觉得你们合得来。只有你们都说好，才算结伴成功。",
		"같은 도시에서 잘 맞을 사람을 찾고, 왜 잘 어울릴지 설명해 줘요. 두 사람 모두 좋다고 해야 함께하게 돼요.",
		"同じ街であなたに合う人を探し、なぜ気が合いそうかを説明します。ふたりともイエスのときだけ一緒に行けます。"},
	{"It's only a match when you both say yes.", "You team up only when you both say yes.",
		"只有你们都说好，才算结伴成功。", "두 사람 모두 좋다고 해야 함께하게 돼요.", "ふたりともイエスのときだけ一緒に行けます。"},
	{"Chat with your match directly. She can suggest an opener or a question when things go quiet, and you can turn that off.",
		"Chat with them directly. She can suggest an opener or a question when things go quiet, and you can turn that off.",
		"直接和搭子聊天。冷场时她可以建议开场白或问题，你也可以关闭这项功能。",
		"상대와 직접 대화해요. 대화가 뜸해지면 첫마디나 질문을 제안해 주고, 이 기능은 끌 수 있어요.",
		"相手と直接チャットします。会話が止まったら話題や質問を提案してくれます。この機能はオフにできます。"},
	{"Plan the date", "Plan the meetup", "安排见面", "만남 계획하기", "会う予定を立てる"},
	{"She shortlists public places based on what you both enjoy. You each pick a favorite privately, and the date is set once you both confirm.",
		"She shortlists public places based on what you both enjoy. You each pick a favorite privately, and the plan is set once you both confirm.",
		"她会根据你们共同的喜好筛选公共场所。你们各自私下选出最喜欢的地方，双方确认后见面就定下来了。",
		"두 사람이 좋아하는 것을 바탕으로 공공장소를 추려 줘요. 각자 마음에 드는 곳을 비공개로 고르고, 둘 다 확인하면 약속이 정해져요.",
		"ふたりの好みに合わせて公共の場所を絞り込みます。それぞれ非公開でお気に入りを選び、ふたりが確認すると予定が決まります。"},
	{"What you share after a date helps her make better introductions next time.",
		"What you share after meeting up helps her make better introductions next time.",
		"见面后你分享的内容，能帮她下次做出更好的介绍。",
		"만난 뒤 들려준 이야기는 다음 소개를 더 좋게 만들어 줘요.",
		"会った後に話してくれたことが、次の紹介をより良くします。"},
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

// Sentences only the China page has.
var cnDownloadText = []cnChange{
	{"Use SynAgent in WeChat or in your browser. For adults 18 and over. Your first introduction is free. After that, messages to Suki use credits, and chatting with the people you team up with stays free.", "",
		"在微信或浏览器中使用 SynAgent。仅限 18 岁及以上。第一次介绍免费，之后和 Suki 对话消耗积分，和搭子聊天一直免费。",
		"위챗이나 브라우저에서 SynAgent를 이용하세요. 18세 이상만 이용할 수 있어요. 첫 소개는 무료이고, 그 후 Suki에게 보내는 메시지에는 크레딧이 들어요. 함께하는 사람과의 채팅은 계속 무료예요.",
		"WeChat かブラウザで SynAgent を使えます。18 歳以上の方が対象です。最初の紹介は無料で、その後 Suki へのメッセージにはクレジットを使います。一緒に行く人とのチャットはずっと無料です。"},
	{"In WeChat", "", "在微信里", "위챗에서", "WeChat で"},
	{"Search for HelloSuki in WeChat, or open the mini program from a friend's share.", "",
		"在微信中搜索 HelloSuki，或从朋友分享的小程序打开。", "위챗에서 HelloSuki를 검색하거나, 친구가 공유한 미니 프로그램을 열어 주세요.", "WeChat で HelloSuki を検索するか、友だちが共有したミニプログラムから開いてください。"},
	{"Nothing to install, and it works on a computer too. Your account is the same one the mini program uses.", "",
		"无需安装，电脑上也能用。账号和小程序是同一个。", "설치할 필요 없고 컴퓨터에서도 돼요. 계정은 미니 프로그램과 같아요.", "インストール不要で、パソコンでも使えます。アカウントはミニプログラムと同じです。"},
	{"Take her with you", "", "随时找她", "언제든 함께", "いつでも一緒に"},
	{"Open SynAgent", "", "打开 SynAgent", "SynAgent 열기", "SynAgent を開く"},
	{"Write to us", "", "联系我们", "문의하기", "お問い合わせ"},
}

var cnDownloadSection = regexp.MustCompile(`(?s)<section class="solid" id="download".*?</section>`)

// chinaLanding rewrites one of the home page's files for the China copy.
func chinaLanding(name string, body []byte) []byte {
	s := string(body)
	switch name {
	case "landing/index.html":
		s = cnDownloadSection.ReplaceAllLiteralString(s, cnDownload)
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
	all := append([]cnChange{}, cnText...)
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
var i18nRow = regexp.MustCompile(`^(\s*)('(?:[^'\\]|\\.)*'|"(?:[^"\\]|\\.)*"|[A-Za-z_][A-Za-z0-9_]*)\s*:\s*\[.*\],?\s*$`)

func jsKey(raw string) string {
	if strings.HasPrefix(raw, "'") || strings.HasPrefix(raw, `"`) {
		return strings.ReplaceAll(strings.ReplaceAll(raw[1:len(raw)-1], `\'`, "'"), `\"`, `"`)
	}
	return raw
}

func jsString(s string) string { b, _ := json.Marshal(s); return string(b) }

func (c cnChange) row(key string) string {
	return "  " + jsString(key) + ": [" + jsString(c.zh) + ", " + jsString(c.ko) + ", " + jsString(c.ja) + "],"
}

// chinaTranslations gives the changed sentences their China wording in every language,
// renames the rows whose English changed, and drops rows only the US page uses.
func chinaTranslations(s string) string {
	byEn := map[string]cnChange{}
	for _, c := range append(append([]cnChange{}, cnText...), cnDownloadText...) {
		byEn[c.en] = c
		if c.newEn != "" {
			byEn[c.newEn] = cnChange{c.newEn, c.newEn, c.zh, c.ko, c.ja}
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
		key := jsKey(m[2])
		if c, ok := byEn[key]; ok {
			if c.newEn != "" {
				key = c.newEn
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, c.row(key))
			continue
		}
		if CNBanned.MatchString(line) {
			continue // only for parts of the page China doesn't show
		}
		seen[key] = true
		out = append(out, line)
	}
	var extra []string
	for _, c := range append(append([]cnChange{}, cnText...), cnDownloadText...) {
		key := c.newEn
		if key == "" {
			key = c.en
		}
		if !seen[key] {
			extra = append(extra, c.row(key))
			seen[key] = true
		}
	}
	joined := strings.Join(out, "\n")
	if len(extra) > 0 {
		joined = strings.Replace(joined, "\n};", "\n"+strings.Join(extra, "\n")+"\n};", 1)
	}
	return joined
}
