package billing

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/damonleelcx/ai-social-network/internal/externalauth"
)

// XPay is WeChat's mini program virtual payment (小程序虚拟支付), which WeChat requires for
// virtual goods such as credits: it closed ordinary WeChat Pay for them on Android. Each credit
// pack is an item (道具) in the virtual payment console, bought directly (short_series_goods).
//
// The flow: the mini program asks for an order with a fresh wx.login code; the server records a
// pending purchase and signs the order with the AppKey (paySig) and the session_key from that
// login (signature); wx.requestVirtualPayment takes the payment; WeChat then pushes
// xpay_goods_deliver_notify to the mini program's message push URL, and the credits are added
// once. If that push is late, the mini program asks the server to check the order with WeChat.
type XPay struct {
	AppID     string
	AppSecret string
	OfferID   string
	// AppKey is the key for Env: the sandbox key in the sandbox, the live key in production.
	AppKey string
	Env    int // 0 live, 1 sandbox
	// PushToken and PushAESKey are the message push settings. The AES key is empty when the
	// push is set to plain text.
	PushToken  string
	PushAESKey []byte
	// APIURL is for tests. Empty means WeChat itself.
	APIURL string

	mu       sync.Mutex
	token    string
	tokenEnd time.Time
}

// XPayConfig is the settings as they come from the environment.
type XPayConfig struct {
	OfferID, AppKey, Env, PushToken, PushAESKey string
}

// NewXPay returns nil when virtual payment isn't configured, and an error when only part is.
func NewXPay(appID, appSecret string, c XPayConfig) (*XPay, error) {
	if c.OfferID == "" && c.AppKey == "" {
		return nil, nil
	}
	if c.OfferID == "" || c.AppKey == "" || c.PushToken == "" || appID == "" || appSecret == "" {
		return nil, errors.New("wechat virtual payment: some settings are missing")
	}
	x := &XPay{AppID: appID, AppSecret: appSecret, OfferID: c.OfferID, AppKey: c.AppKey, PushToken: c.PushToken}
	switch c.Env {
	case "", "0":
	case "1":
		x.Env = 1
	default:
		return nil, errors.New("wechat virtual payment: env must be 0 (live) or 1 (sandbox)")
	}
	if c.PushAESKey != "" {
		key, err := base64.StdEncoding.DecodeString(c.PushAESKey + "=")
		if err != nil || len(key) != 32 {
			return nil, errors.New("wechat virtual payment: the message push EncodingAESKey must be 43 characters")
		}
		x.PushAESKey = key
	}
	return x, nil
}

// XPayProduct is the item id of a credit pack in the virtual payment console.
func XPayProduct(p Package) string { return fmt.Sprintf("credits_%d", p.Credits) }

func hmacHex(key, msg string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(msg))
	return hex.EncodeToString(h.Sum(nil))
}

// VirtualOrder is what wx.requestVirtualPayment takes, and the purchase to ask about afterwards.
type VirtualOrder struct {
	Mode      string `json:"mode"`
	SignData  string `json:"signData"`
	PaySig    string `json:"paySig"`
	Signature string `json:"signature"`
	Order     string `json:"order"`
}

func packageByID(id string) *Package {
	for i := range WeChatPackages {
		if WeChatPackages[i].ID == id {
			return &WeChatPackages[i]
		}
	}
	return nil
}

// XPayOrder starts a virtual payment purchase. The openid and session_key are used for this
// order and not kept.
func (s *Service) XPayOrder(ctx context.Context, userID, packageID, code string) (*VirtualOrder, error) {
	x := s.XPay
	if x == nil {
		return nil, &UserError{"Buying credits isn't open yet."}
	}
	pkg := packageByID(packageID)
	if pkg == nil {
		return nil, &UserError{"Pick one of the credit packs."}
	}
	_, sessionKey, err := (&externalauth.WeChat{AppID: x.AppID, AppSecret: x.AppSecret, URL: x.APIURL, HTTP: s.HTTP}).Session(ctx, code)
	if errors.Is(err, externalauth.ErrRejected) || err == nil && sessionKey == "" {
		return nil, &UserError{"WeChat couldn't confirm who you are. Please try again."}
	}
	if err != nil {
		return nil, err
	}
	var id string
	if err := s.Pool.QueryRow(ctx, `INSERT INTO credit_purchases (user_id, package, credits, amount_cents, currency) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		userID, pkg.ID, pkg.Credits, pkg.AmountCents, pkg.Currency).Scan(&id); err != nil {
		return nil, err
	}
	order := tradeNo(id)
	// Field order and spacing must reach WeChat exactly as signed, so the string is built once
	// here and the mini program passes it through untouched.
	data, err := json.Marshal(struct {
		OfferID      string `json:"offerId"`
		BuyQuantity  int    `json:"buyQuantity"`
		Env          int    `json:"env"`
		CurrencyType string `json:"currencyType"`
		ProductID    string `json:"productId"`
		GoodsPrice   int    `json:"goodsPrice"`
		OutTradeNo   string `json:"outTradeNo"`
		Attach       string `json:"attach"`
	}{x.OfferID, 1, x.Env, "CNY", XPayProduct(*pkg), pkg.AmountCents, order, pkg.ID})
	if err != nil {
		return nil, err
	}
	signData := string(data)
	return &VirtualOrder{
		Mode:      "short_series_goods",
		SignData:  signData,
		PaySig:    hmacHex(x.AppKey, "requestVirtualPayment&"+signData),
		Signature: hmacHex(sessionKey, signData),
		Order:     order,
	}, nil
}

// ── Message push: the delivery notice ───────────────────────────────────────

// ErrBadPush is a message push that didn't come from WeChat.
var ErrBadPush = errors.New("message push: bad signature")

func sha1Sorted(parts ...string) string {
	sort.Strings(parts)
	sum := sha1.Sum([]byte(strings.Join(parts, "")))
	return hex.EncodeToString(sum[:])
}

func sameText(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

// VerifyPushURL answers WeChat's check of the message push URL (GET with echostr).
func (x *XPay) VerifyPushURL(signature, timestamp, nonce string) bool {
	return sameText(signature, sha1Sorted(x.PushToken, timestamp, nonce))
}

// openPush checks a pushed message and, in safe mode, decrypts it. q holds the URL's query.
func (x *XPay) openPush(body []byte, q map[string]string) ([]byte, error) {
	if q["encrypt_type"] != "aes" {
		if !sameText(q["signature"], sha1Sorted(x.PushToken, q["timestamp"], q["nonce"])) {
			return nil, ErrBadPush
		}
		return body, nil
	}
	if len(x.PushAESKey) != 32 {
		return nil, errors.New("message push: safe mode is on, but no EncodingAESKey is set")
	}
	var env struct {
		Encrypt string `json:"Encrypt"`
	}
	if err := json.Unmarshal(body, &env); err != nil || env.Encrypt == "" {
		return nil, ErrBadPush
	}
	if !sameText(q["msg_signature"], sha1Sorted(x.PushToken, q["timestamp"], q["nonce"], env.Encrypt)) {
		return nil, ErrBadPush
	}
	raw, err := base64.StdEncoding.DecodeString(env.Encrypt)
	if err != nil || len(raw) < aes.BlockSize || len(raw)%aes.BlockSize != 0 {
		return nil, ErrBadPush
	}
	block, err := aes.NewCipher(x.PushAESKey)
	if err != nil {
		return nil, err
	}
	plain := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, x.PushAESKey[:16]).CryptBlocks(plain, raw)
	pad := int(plain[len(plain)-1])
	if pad < 1 || pad > 32 || pad > len(plain) {
		return nil, ErrBadPush
	}
	plain = plain[:len(plain)-pad]
	// 16 random bytes, a 4-byte length, the message, then the AppID it was sent to.
	if len(plain) < 20 {
		return nil, ErrBadPush
	}
	n := int(binary.BigEndian.Uint32(plain[16:20]))
	if n < 0 || 20+n > len(plain) || string(plain[20+n:]) != x.AppID {
		return nil, ErrBadPush
	}
	return plain[20 : 20+n], nil
}

// HandleXPayPush takes a message push. Only virtual payment delivery notices matter; anything
// else is acknowledged and ignored. An error asks WeChat to send it again.
func (s *Service) HandleXPayPush(ctx context.Context, body []byte, q map[string]string) (*Paid, error) {
	x := s.XPay
	if x == nil {
		return nil, ErrBadPush
	}
	msg, err := x.openPush(body, q)
	if err != nil {
		return nil, err
	}
	var ev struct {
		Event         string `json:"Event"`
		OpenID        string `json:"openid"`
		OutTradeNo    string `json:"OutTradeNo"`
		Env           int    `json:"Env"`
		WechatPayInfo *struct {
			MchOrderNo    string `json:"MchOrderNo"`
			TransactionID string `json:"TransactionId"`
		} `json:"WechatPayInfo"`
		GoodsInfo *struct {
			ProductID string `json:"ProductId"`
			Quantity  int    `json:"Quantity"`
			OrigPrice int    `json:"OrigPrice"`
		} `json:"GoodsInfo"`
	}
	if err := json.Unmarshal(msg, &ev); err != nil {
		return nil, ErrBadPush
	}
	if ev.Event != "xpay_goods_deliver_notify" {
		return nil, nil
	}
	if ev.Env != x.Env {
		// A sandbox notice reaching a live server, or the other way round: never credit it.
		slog.Warn("virtual payment notice for the other environment", "env", ev.Env)
		return nil, nil
	}
	if ev.GoodsInfo == nil {
		return nil, errors.New("virtual payment notice without goods")
	}
	paid := ev.GoodsInfo.OrigPrice * max(ev.GoodsInfo.Quantity, 1)
	if err := s.checkProduct(ctx, ev.OutTradeNo, ev.GoodsInfo.ProductID); err != nil {
		return nil, err
	}
	ref := "xpay:" + ev.OutTradeNo
	if ev.WechatPayInfo != nil && ev.WechatPayInfo.TransactionID != "" {
		ref = ev.WechatPayInfo.TransactionID
	}
	return s.completePurchase(ctx, ev.OutTradeNo, ref, paid, "cny")
}

// checkProduct makes sure the item paid for is the pack this purchase is for.
func (s *Service) checkProduct(ctx context.Context, order, productID string) error {
	var pkgID string
	err := s.Pool.QueryRow(ctx, `SELECT package FROM credit_purchases WHERE replace(id::text,'-','')=$1`, order).Scan(&pkgID)
	if err != nil {
		return nil // completePurchase reports an unknown purchase
	}
	if p := packageByID(pkgID); p == nil || XPayProduct(*p) != productID {
		return fmt.Errorf("virtual payment %s: paid for %q, expected pack %q", order, productID, pkgID)
	}
	return nil
}

// ── Server API: checking an order when the push is late ─────────────────────

func (x *XPay) api() string {
	if x.APIURL != "" {
		return x.APIURL
	}
	return "https://api.weixin.qq.com"
}

// accessToken is the mini program's stable access token, which doesn't invalidate others.
func (s *Service) accessToken(ctx context.Context) (string, error) {
	x := s.XPay
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.token != "" && time.Now().Before(x.tokenEnd) {
		return x.token, nil
	}
	body, _ := json.Marshal(map[string]string{"grant_type": "client_credential", "appid": x.AppID, "secret": x.AppSecret})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, x.api()+"/cgi-bin/stable_token", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client().Do(req)
	if err != nil {
		return "", errors.New("wechat: can't reach WeChat for an access token")
	}
	defer resp.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&out); err != nil || out.AccessToken == "" {
		return "", fmt.Errorf("wechat: no access token (errcode %d)", out.ErrCode)
	}
	x.token, x.tokenEnd = out.AccessToken, time.Now().Add(time.Duration(max(out.ExpiresIn-300, 60))*time.Second)
	return x.token, nil
}

// xpayCall posts to a virtual payment server API, signed with the AppKey.
func (s *Service) xpayCall(ctx context.Context, path string, in, out any) error {
	x := s.XPay
	tok, err := s.accessToken(ctx)
	if err != nil {
		return err
	}
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	u := x.api() + path + "?access_token=" + tok + "&pay_sig=" + hmacHex(x.AppKey, path+"&"+string(body))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client().Do(req)
	if err != nil {
		return errors.New("wechat: can't reach WeChat") // the URL holds the access token
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	var e struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(raw, &e)
	if e.ErrCode != 0 {
		return fmt.Errorf("wechat %s: errcode %d %s", path, e.ErrCode, e.ErrMsg)
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

// CheckXPayOrder asks WeChat about one of this person's purchases, and adds the credits if it
// was paid and the delivery notice hasn't arrived. Safe to call any number of times.
func (s *Service) CheckXPayOrder(ctx context.Context, userID, order string) (*Paid, error) {
	if s.XPay == nil {
		return nil, &UserError{"Buying credits isn't open yet."}
	}
	var status, pkgID string
	err := s.Pool.QueryRow(ctx, `SELECT status, package FROM credit_purchases WHERE replace(id::text,'-','')=$1 AND user_id=$2`, order, userID).Scan(&status, &pkgID)
	if err != nil {
		return nil, &UserError{"That purchase isn't yours."}
	}
	if status == "paid" {
		return nil, nil
	}
	var out struct {
		Order *struct {
			Status    int    `json:"status"`
			OrderFee  int    `json:"order_fee"`
			WxOrderID string `json:"wx_order_id"`
			ProductID string `json:"product_id"`
		} `json:"order"`
	}
	if err := s.xpayCall(ctx, "/xpay/query_order", map[string]any{"order_id": order, "env": s.XPay.Env}, &out); err != nil {
		return nil, err
	}
	o := out.Order
	// 2 paid and waiting for delivery, 3 delivering, 4 delivered.
	if o == nil || o.Status < 2 || o.Status > 4 {
		return nil, nil
	}
	p := packageByID(pkgID)
	if p == nil {
		return nil, fmt.Errorf("virtual payment %s: unknown pack %q", order, pkgID)
	}
	paid, err := s.completePurchase(ctx, order, "xpay:"+o.WxOrderID, p.AmountCents, "cny")
	if err != nil {
		return nil, err
	}
	if o.Status == 2 {
		// Tell WeChat it's delivered, so it stops retrying the notice and the order shows done.
		if err := s.xpayCall(ctx, "/xpay/notify_provide_goods", map[string]any{"order_id": order, "env": s.XPay.Env}, nil); err != nil {
			slog.Warn("virtual payment delivered but not reported", "order", order, "err", err)
		}
	}
	return paid, nil
}

// PushQuery turns the push URL's query into the map HandleXPayPush takes.
func PushQuery(get func(string) string) map[string]string {
	q := map[string]string{}
	for _, k := range []string{"signature", "timestamp", "nonce", "encrypt_type", "msg_signature"} {
		q[k] = get(k)
	}
	return q
}
