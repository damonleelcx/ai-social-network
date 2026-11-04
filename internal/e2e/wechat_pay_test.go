package e2e

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/damonleelcx/ai-social-network/internal/billing"
)

const testV3Key = "0123456789abcdef0123456789abcdef"

// A stand-in for WeChat: it checks the merchant's signature the way WeChat Pay does, and
// signs and encrypts notifications with its own key.
func TestWeChatPayAddsCredits(t *testing.T) {
	e := setup(t)
	merchant, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	platform, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var order map[string]any
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sns/jscode2session" {
			if r.URL.Query().Get("js_code") != "good-code" {
				_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 40029})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"openid": "openid-ada"})
			return
		}
		raw, _ := io.ReadAll(r.Body)
		h := r.Header.Get("Authorization")
		get := func(k string) string {
			m := regexp.MustCompile(k + `="([^"]*)"`).FindStringSubmatch(h)
			if m == nil {
				return ""
			}
			return m[1]
		}
		sig, _ := base64.StdEncoding.DecodeString(get("signature"))
		sum := sha256.Sum256([]byte(r.Method + "\n" + r.URL.Path + "\n" + get("timestamp") + "\n" + get("nonce_str") + "\n" + string(raw) + "\n"))
		if !strings.HasPrefix(h, "WECHATPAY2-SHA256-RSA2048 ") || rsa.VerifyPKCS1v15(&merchant.PublicKey, crypto.SHA256, sum[:], sig) != nil {
			w.WriteHeader(401)
			return
		}
		_ = json.Unmarshal(raw, &order)
		_ = json.NewEncoder(w).Encode(map[string]string{"prepay_id": "wx-prepay-1"})
	}))
	defer fake.Close()
	e.billing.WeChat = &billing.WeChat{AppID: "wxapp", AppSecret: "secret", MchID: "mch1", MchSerial: "SER", APIv3Key: testV3Key,
		PublicKeyID: "PUB_KEY_ID_1", PrivateKey: merchant, PublicKey: &platform.PublicKey, PayURL: fake.URL, AuthURL: fake.URL}

	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()

	var st billing.Status
	a.do("GET", "/v1/credits", nil, 200, &st)
	if !st.WeChatEnabled || len(st.WeChatPackages) != 3 || st.PaymentsEnabled {
		t.Fatalf("status: %+v", st)
	}

	// A bad login code and an unknown pack are refused.
	a.do("POST", "/v1/credits/wechat/prepay", map[string]any{"package": "small", "code": "bad"}, 422, nil)
	a.do("POST", "/v1/credits/wechat/prepay", map[string]any{"package": "nope", "code": "good-code"}, 422, nil)

	var p billing.PrepayParams
	a.do("POST", "/v1/credits/wechat/prepay", map[string]any{"package": "small", "code": "good-code"}, 200, &p)
	if p.Package != "prepay_id=wx-prepay-1" || p.SignType != "RSA" {
		t.Fatalf("prepay: %+v", p)
	}
	sum := sha256.Sum256([]byte("wxapp\n" + p.TimeStamp + "\n" + p.NonceStr + "\n" + p.Package + "\n"))
	paySig, _ := base64.StdEncoding.DecodeString(p.PaySign)
	if err := rsa.VerifyPKCS1v15(&merchant.PublicKey, crypto.SHA256, sum[:], paySig); err != nil {
		t.Fatal("paySign doesn't verify:", err)
	}
	amount := order["amount"].(map[string]any)
	if amount["total"].(float64) != 3500 || amount["currency"] != "CNY" || order["payer"].(map[string]any)["openid"] != "openid-ada" {
		t.Fatalf("order: %v", order)
	}
	no := order["out_trade_no"].(string)
	if len(no) > 32 || strings.Contains(no, "-") {
		t.Fatalf("out_trade_no: %q", no)
	}

	notify := func(key *rsa.PrivateKey, serial string, total int, ts time.Time) int {
		plain, _ := json.Marshal(map[string]any{"out_trade_no": no, "transaction_id": "tx1", "trade_state": "SUCCESS", "amount": map[string]any{"total": total, "currency": "CNY"}})
		block, _ := aes.NewCipher([]byte(testV3Key))
		gcm, _ := cipher.NewGCM(block)
		nonce := "abcdefghijkl"
		body, _ := json.Marshal(map[string]any{"event_type": "TRANSACTION.SUCCESS", "resource": map[string]any{
			"algorithm": "AEAD_AES_256_GCM", "nonce": nonce, "associated_data": "transaction",
			"ciphertext": base64.StdEncoding.EncodeToString(gcm.Seal(nil, []byte(nonce), plain, []byte("transaction")))}})
		stamp := fmt.Sprint(ts.Unix())
		digest := sha256.Sum256([]byte(stamp + "\nn1\n" + string(body) + "\n"))
		sig, _ := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
		req, _ := http.NewRequest("POST", e.api.URL+"/v1/billing/wechat", strings.NewReader(string(body)))
		req.Header.Set("Wechatpay-Timestamp", stamp)
		req.Header.Set("Wechatpay-Nonce", "n1")
		req.Header.Set("Wechatpay-Serial", serial)
		req.Header.Set("Wechatpay-Signature", base64.StdEncoding.EncodeToString(sig))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	balance := func() int {
		a.do("GET", "/v1/credits", nil, 200, &st)
		return st.Balance
	}

	// Forged, the wrong key id, stale, or the wrong amount: nothing.
	if c := notify(merchant, "PUB_KEY_ID_1", 3500, time.Now()); c != 400 {
		t.Fatalf("forged: %d", c)
	}
	if c := notify(platform, "OTHER", 3500, time.Now()); c != 400 {
		t.Fatalf("wrong key id: %d", c)
	}
	if c := notify(platform, "PUB_KEY_ID_1", 3500, time.Now().Add(-time.Hour)); c != 400 {
		t.Fatalf("stale: %d", c)
	}
	if c := notify(platform, "PUB_KEY_ID_1", 1, time.Now()); c < 300 {
		t.Fatal("a wrong amount was accepted")
	}
	if balance() != 0 {
		t.Fatal("credits from a bad notification")
	}

	// Paid, and delivered twice: credited once, and Suki says so once.
	for i := 0; i < 2; i++ {
		if c := notify(platform, "PUB_KEY_ID_1", 3500, time.Now()); c != 204 {
			t.Fatalf("notify: %d", c)
		}
	}
	if balance() != 100 {
		t.Fatalf("balance: %d", st.Balance)
	}
	var paid string
	if err := e.pool.QueryRow(context.Background(), `SELECT status FROM credit_purchases WHERE user_id=$1 AND status='paid'`, a.id).Scan(&paid); err != nil {
		t.Fatal(err)
	}
}
