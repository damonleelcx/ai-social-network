package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func chinaGet(t *testing.T, path string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	(&Server{Region: "cn"}).Handler().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	if rec.Code != 200 {
		t.Fatalf("%s: %d", path, rec.Code)
	}
	return rec.Body.String()
}

func TestChinaSiteSaysNothingItShouldnt(t *testing.T) {
	paths := []string{"/", "/landing/app.js", "/landing/i18n.js", "/landing/scene.js", "/open"}
	for _, page := range []string{"terms", "privacy", "contact", "delete-account", "child-safety"} {
		paths = append(paths, "/legal/"+page, "/legal/"+page+"?lang=en", "/legal/"+page+"?lang=ko", "/legal/"+page+"?lang=ja", "/legal/"+page+"?lang=es")
	}
	if got := chinaGet(t, "/legal/privacy"); !strings.Contains(got, "南京知遇无界科技有限公司") || !strings.Contains(got, "中华人民共和国境内") {
		t.Error("China privacy policy isn't China's")
	}
	for _, path := range paths {
		body := chinaGet(t, path)
		body = strings.NewReplacer(".matches", "", "Date.now", "", "new Date", "").Replace(body) // code, not words
		for _, m := range CNBanned.FindAllStringIndex(body, -1) {
			lo, hi := max(m[0]-60, 0), min(m[1]+60, len(body))
			t.Errorf("%s says %q: …%s…", path, body[m[0]:m[1]], body[lo:hi])
		}
	}
	page := chinaGet(t, "/")
	for _, want := range []string{"In WeChat", "mailto:support@zhiyuwujie.top", `<option value="zh">`, "An assistant you talk to,"} {
		if !strings.Contains(page, want) {
			t.Errorf("China home page lacks %q", want)
		}
	}
	if !strings.Contains(page, `value="ko"`) || !strings.Contains(page, `value="ja"`) {
		t.Error("China home page should offer Korean and Japanese")
	}
	i18n := chinaGet(t, "/landing/i18n.js")
	for _, want := range []string{`"An assistant you talk to,": ["一位能聊天的找搭子助手，", "끝없는 목록이 아니라,"`, `"In WeChat": ["在微信里", "위챗에서", "WeChat で"]`, "18 岁", "18세", "18歳"} {
		if !strings.Contains(i18n, want) {
			t.Errorf("China translations lack %q", want)
		}
	}
	// The US site is untouched.
	rec := httptest.NewRecorder()
	(&Server{}).Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), "SynAgent.apk") {
		t.Error("the US home page changed")
	}
}
