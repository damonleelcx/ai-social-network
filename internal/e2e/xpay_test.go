package e2e

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/damonleelcx/ai-social-network/internal/billing"
)

func hmacOf(key, msg string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(msg))
	return hex.EncodeToString(h.Sum(nil))
}

func sha1Of(parts ...string) string {
	sort.Strings(parts)
	sum := sha1.Sum([]byte(strings.Join(parts, "")))
	return hex.EncodeToString(sum[:])
}

// Credits through WeChat's virtual payment, against a stand-in for WeChat: the order is signed
// the way wx.requestVirtualPayment checks it, the delivery push (plain and encrypted) adds the
// credits once, and a late push is covered by asking WeChat about the order.
func TestWeChatVirtualPaymentAddsCredits(t *testing.T) {
	e := setup(t)
	const appKey, sessionKey, token = "app-key-123", "session-key-abc", "push-token"
	aesKey := []byte("0123456789abcdef0123456789abcdef")
	var mu sync.Mutex
	orders := map[string]int{} // order id -> WeChat's status
	var delivered []string
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sns/jscode2session":
			if r.URL.Query().Get("js_code") != "good-code" {
				_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 40029})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"openid": "openid-ada", "session_key": sessionKey})
		case "/cgi-bin/stable_token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at-1", "expires_in": 7200})
		case "/xpay/query_order", "/xpay/notify_provide_goods":
			raw, _ := io.ReadAll(r.Body)
			if r.URL.Query().Get("access_token") != "at-1" || r.URL.Query().Get("pay_sig") != hmacOf(appKey, r.URL.Path+"&"+string(raw)) {
				_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 268490003, "errmsg": "sign error"})
				return
			}
			var in struct {
				OrderID string `json:"order_id"`
				Env     int    `json:"env"`
			}
			_ = json.Unmarshal(raw, &in)
			mu.Lock()
			defer mu.Unlock()
			if r.URL.Path == "/xpay/notify_provide_goods" {
				delivered = append(delivered, in.OrderID)
				orders[in.OrderID] = 4
				_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 0})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 0, "order": map[string]any{
				"order_id": in.OrderID, "status": orders[in.OrderID], "order_fee": 8800, "wx_order_id": "wx-" + in.OrderID}})
		default:
			w.WriteHeader(404)
		}
	}))
	defer fake.Close()
	x, err := billing.NewXPay("wxapp", "secret", billing.XPayConfig{OfferID: "1450000001", AppKey: appKey, Env: "1", PushToken: token,
		PushAESKey: strings.TrimRight(base64.StdEncoding.EncodeToString(aesKey), "=")})
	if err != nil {
		t.Fatal(err)
	}
	x.APIURL = fake.URL
	e.billing.XPay = x

	a := e.signup(t, "Ada", "ada@example.com")
	a.verify()
	var st billing.Status
	balance := func() int {
		a.do("GET", "/v1/credits", nil, 200, &st)
		return st.Balance
	}
	if balance() != 0 || !st.WeChatEnabled || !st.WeChatVirtual {
		t.Fatalf("status: %+v", st)
	}

	a.do("POST", "/v1/credits/wechat/virtual", map[string]any{"package": "small", "code": "bad"}, 422, nil)
	a.do("POST", "/v1/credits/wechat/virtual", map[string]any{"package": "nope", "code": "good-code"}, 422, nil)

	// The order is signed exactly as WeChat checks it.
	var o billing.VirtualOrder
	a.do("POST", "/v1/credits/wechat/virtual", map[string]any{"package": "small", "code": "good-code"}, 200, &o)
	if o.Mode != "short_series_goods" || o.PaySig != hmacOf(appKey, "requestVirtualPayment&"+o.SignData) || o.Signature != hmacOf(sessionKey, o.SignData) {
		t.Fatalf("order signatures: %+v", o)
	}
	var sd map[string]any
	if err := json.Unmarshal([]byte(o.SignData), &sd); err != nil {
		t.Fatal(err)
	}
	if sd["offerId"] != "1450000001" || sd["productId"] != "credits_100" || sd["goodsPrice"].(float64) != 3500 || sd["env"].(float64) != 1 ||
		sd["outTradeNo"] != o.Order || sd["currencyType"] != "CNY" || len(o.Order) > 32 || strings.Contains(o.Order, "-") {
		t.Fatalf("signData: %s", o.SignData)
	}

	push := func(event map[string]any, encrypt bool, forge bool) (int, string) {
		body, _ := json.Marshal(event)
		ts, nonce := fmt.Sprint(time.Now().Unix()), "nonce1"
		q := url.Values{"timestamp": {ts}, "nonce": {nonce}}
		if encrypt {
			msg := append([]byte("0123456789abcdef"), make([]byte, 4)...)
			binary.BigEndian.PutUint32(msg[16:20], uint32(len(body)))
			msg = append(append(msg, body...), "wxapp"...)
			pad := 32 - len(msg)%32
			msg = append(msg, bytes.Repeat([]byte{byte(pad)}, pad)...)
			block, _ := aes.NewCipher(aesKey)
			out := make([]byte, len(msg))
			cipher.NewCBCEncrypter(block, aesKey[:16]).CryptBlocks(out, msg)
			enc := base64.StdEncoding.EncodeToString(out)
			body, _ = json.Marshal(map[string]string{"ToUserName": "gh_1", "Encrypt": enc})
			q.Set("encrypt_type", "aes")
			q.Set("msg_signature", sha1Of(token, ts, nonce, enc))
		}
		sig := sha1Of(token, ts, nonce)
		if forge {
			sig, q["msg_signature"] = "0000", []string{"0000"}
		}
		q.Set("signature", sig)
		resp, err := http.Post(e.api.URL+"/v1/billing/wechat/push?"+q.Encode(), "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(raw)
	}
	deliver := func(order, product string, price, env int) map[string]any {
		return map[string]any{"ToUserName": "gh_1", "FromUserName": "wx-official", "MsgType": "event", "Event": "xpay_goods_deliver_notify",
			"openid": "openid-ada", "OutTradeNo": order, "Env": env,
			"WechatPayInfo": map[string]any{"MchOrderNo": "m1", "TransactionId": "tx-" + order},
			"GoodsInfo":     map[string]any{"ProductId": product, "Quantity": 1, "OrigPrice": price, "ActualPrice": price, "Attach": "small"}}
	}

	// The push URL check WeChat makes when it's saved in the console.
	q := url.Values{"timestamp": {"1"}, "nonce": {"n"}, "echostr": {"hello"}, "signature": {sha1Of(token, "1", "n")}}
	resp, err := http.Get(e.api.URL + "/v1/billing/wechat/push?" + q.Encode())
	if err != nil {
		t.Fatal(err)
	}
	echo, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != 200 || string(echo) != "hello" {
		t.Fatalf("url check: %d %q", resp.StatusCode, echo)
	}

	// Forged, the wrong item, the live environment or the wrong price: nothing.
	if c, _ := push(deliver(o.Order, "credits_100", 3500, 1), false, true); c != 403 {
		t.Fatalf("forged: %d", c)
	}
	if c, _ := push(deliver(o.Order, "credits_100", 3500, 1), true, true); c != 403 {
		t.Fatalf("forged encrypted: %d", c)
	}
	if _, body := push(deliver(o.Order, "credits_1000", 3500, 1), false, false); !strings.Contains(body, `"ErrCode":1`) {
		t.Fatalf("wrong item: %s", body)
	}
	if _, body := push(deliver(o.Order, "credits_100", 3500, 0), false, false); !strings.Contains(body, `"ErrCode":0`) {
		t.Fatalf("other environment: %s", body)
	}
	if _, body := push(deliver(o.Order, "credits_100", 1, 1), false, false); !strings.Contains(body, `"ErrCode":1`) {
		t.Fatalf("wrong price: %s", body)
	}
	if balance() != 0 {
		t.Fatal("credits from a bad notice")
	}
	// Something else pushed to the same URL is acknowledged and ignored.
	if _, body := push(map[string]any{"MsgType": "event", "Event": "user_enter_tempsession"}, false, false); !strings.Contains(body, `"ErrCode":0`) {
		t.Fatalf("other event: %s", body)
	}

	// Delivered, encrypted, twice: credited once.
	for i := 0; i < 2; i++ {
		if c, body := push(deliver(o.Order, "credits_100", 3500, 1), true, false); c != 200 || !strings.Contains(body, `"ErrCode":0`) {
			t.Fatalf("deliver: %d %s", c, body)
		}
	}
	if balance() != 100 {
		t.Fatalf("balance: %d", st.Balance)
	}

	// The push never comes for the second order: asking WeChat about it adds the credits and
	// reports the delivery. Not paid yet, someone else's order, or asked again: nothing more.
	var o2 billing.VirtualOrder
	a.do("POST", "/v1/credits/wechat/virtual", map[string]any{"package": "medium", "code": "good-code"}, 200, &o2)
	var check struct {
		Paid bool `json:"paid"`
	}
	a.do("POST", "/v1/credits/wechat/virtual/check", map[string]any{"order": o2.Order}, 200, &check)
	if check.Paid || balance() != 100 {
		t.Fatal("credits for an unpaid order")
	}
	mu.Lock()
	orders[o2.Order] = 2
	mu.Unlock()
	b := e.signup(t, "Bo", "bo@example.com")
	b.verify()
	b.do("POST", "/v1/credits/wechat/virtual/check", map[string]any{"order": o2.Order}, 422, nil)
	a.do("POST", "/v1/credits/wechat/virtual/check", map[string]any{"order": o2.Order}, 200, &check)
	a.do("POST", "/v1/credits/wechat/virtual/check", map[string]any{"order": o2.Order}, 200, nil)
	if !check.Paid || balance() != 400 {
		t.Fatalf("after the check: paid %v, balance %d", check.Paid, st.Balance)
	}
	mu.Lock()
	if len(delivered) != 1 || delivered[0] != o2.Order {
		t.Fatalf("delivery reported: %v", delivered)
	}
	mu.Unlock()
	// And when the push does arrive afterwards, it changes nothing.
	if _, body := push(deliver(o2.Order, "credits_300", 8800, 1), false, false); !strings.Contains(body, `"ErrCode":0`) || balance() != 400 {
		t.Fatalf("late push: %s, balance %d", body, st.Balance)
	}
}
