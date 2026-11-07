package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestICPFooterOnlyWhereConfigured(t *testing.T) {
	get := func(s *Server) string {
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		if rec.Code != 200 {
			t.Fatalf("status %d", rec.Code)
		}
		return rec.Body.String()
	}
	if body := get(&Server{}); strings.Contains(body, `class="icp"`) {
		t.Fatal("ICP notice shown without configuration")
	}
	body := get(&Server{ICPFooter: ICPFooter("苏ICP备12345678号-1", "苏B2-20262623")})
	for _, want := range []string{`href="https://beian.miit.gov.cn/"`, "苏ICP备12345678号-1", "增值电信业务经营许可证：苏B2-20262623"} {
		if !strings.Contains(body, want) {
			t.Fatalf("home page lacks %q", want)
		}
	}
	if i, j := strings.Index(body, `class="icp"`), strings.Index(body, "</footer>"); i < 0 || i > j {
		t.Fatal("notice is not inside the footer")
	}
	if ICPFooter("", "") != "" {
		t.Fatal("empty values should give no notice")
	}
}
