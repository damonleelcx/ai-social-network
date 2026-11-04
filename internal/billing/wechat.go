package billing

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// WeChatPackages are what's for sale in the mini program, in fen.
var WeChatPackages = []Package{
	{ID: "small", Credits: 100, AmountCents: 3500, Currency: "cny"},
	{ID: "medium", Credits: 300, AmountCents: 8800, Currency: "cny"},
	{ID: "large", Credits: 1000, AmountCents: 23800, Currency: "cny"},
}

// WeChat is WeChat Pay API v3 (JSAPI) for the mini program, in public-key mode.
type WeChat struct {
	AppID       string
	AppSecret   string
	MchID       string
	MchSerial   string
	APIv3Key    string
	PublicKeyID string
	PrivateKey  *rsa.PrivateKey
	PublicKey   *rsa.PublicKey
	// Tests point these elsewhere. Empty means WeChat itself.
	PayURL  string
	AuthURL string
}

// WeChatConfig is the settings as they come from the environment.
type WeChatConfig struct {
	AppID, AppSecret, MchID, MchSerial, APIv3Key, PrivateKeyPEM, PublicKeyID, PublicKeyPEM string
}

// NewWeChat returns nil when nothing is configured, and an error when only some of it is.
func NewWeChat(c WeChatConfig) (*WeChat, error) {
	all := []string{c.AppID, c.AppSecret, c.MchID, c.MchSerial, c.APIv3Key, c.PrivateKeyPEM, c.PublicKeyID, c.PublicKeyPEM}
	set := 0
	for _, v := range all {
		if v != "" {
			set++
		}
	}
	if set == 0 {
		return nil, nil
	}
	if set != len(all) {
		return nil, errors.New("wechat pay: some settings are missing")
	}
	if len(c.APIv3Key) != 32 {
		return nil, errors.New("wechat pay: the APIv3 key must be 32 bytes")
	}
	block, _ := pem.Decode([]byte(strings.ReplaceAll(c.PrivateKeyPEM, `\n`, "\n")))
	if block == nil {
		return nil, errors.New("wechat pay: the merchant private key isn't PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("wechat pay: the merchant private key can't be read")
	}
	private, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("wechat pay: the merchant private key isn't RSA")
	}
	block, _ = pem.Decode([]byte(strings.ReplaceAll(c.PublicKeyPEM, `\n`, "\n")))
	if block == nil {
		return nil, errors.New("wechat pay: the public key isn't PEM")
	}
	parsedPub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, errors.New("wechat pay: the public key can't be read")
	}
	public, ok := parsedPub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("wechat pay: the public key isn't RSA")
	}
	return &WeChat{AppID: c.AppID, AppSecret: c.AppSecret, MchID: c.MchID, MchSerial: c.MchSerial, APIv3Key: c.APIv3Key,
		PublicKeyID: c.PublicKeyID, PrivateKey: private, PublicKey: public}, nil
}

func (w *WeChat) sign(message string) (string, error) {
	sum := sha256.Sum256([]byte(message))
	sig, err := rsa.SignPKCS1v15(rand.Reader, w.PrivateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func nonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// openID turns the code from wx.login into who this person is to this mini program.
func (s *Service) openID(ctx context.Context, code string) (string, error) {
	w := s.WeChat
	base := w.AuthURL
	if base == "" {
		base = "https://api.weixin.qq.com"
	}
	q := url.Values{"appid": {w.AppID}, "secret": {w.AppSecret}, "js_code": {code}, "grant_type": {"authorization_code"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/sns/jscode2session?"+q.Encode(), nil)
	if err != nil {
		return "", errors.New("wechat login: bad request")
	}
	resp, err := s.client().Do(req)
	if err != nil {
		// This error quotes the URL, and the URL holds the app secret.
		return "", errors.New("wechat login: can't reach WeChat")
	}
	defer resp.Body.Close()
	var out struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&out); err != nil {
		return "", errors.New("wechat login: unreadable answer")
	}
	if out.OpenID == "" {
		return "", &UserError{"WeChat couldn't confirm who you are. Please try again."}
	}
	return out.OpenID, nil
}

func (s *Service) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

// PrepayParams are what wx.requestPayment takes.
type PrepayParams struct {
	TimeStamp string `json:"timeStamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

// tradeNo is a purchase id in the form WeChat accepts: no more than 32 letters and digits.
func tradeNo(id string) string { return strings.ReplaceAll(id, "-", "") }

// WeChatPrepay starts a purchase. The person's openid is used for this order and not kept.
func (s *Service) WeChatPrepay(ctx context.Context, userID, packageID, code string) (*PrepayParams, error) {
	w := s.WeChat
	if w == nil {
		return nil, &UserError{"Buying credits isn't open yet."}
	}
	var pkg *Package
	for i := range WeChatPackages {
		if WeChatPackages[i].ID == packageID {
			pkg = &WeChatPackages[i]
		}
	}
	if pkg == nil {
		return nil, &UserError{"Pick one of the credit packs."}
	}
	openid, err := s.openID(ctx, code)
	if err != nil {
		return nil, err
	}
	var id string
	if err := s.Pool.QueryRow(ctx, `INSERT INTO credit_purchases (user_id, package, credits, amount_cents, currency) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		userID, pkg.ID, pkg.Credits, pkg.AmountCents, pkg.Currency).Scan(&id); err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]any{
		"appid":        w.AppID,
		"mchid":        w.MchID,
		"description":  fmt.Sprintf("SynAgent %d 积分", pkg.Credits),
		"out_trade_no": tradeNo(id),
		"notify_url":   s.PublicOrigin + "/v1/billing/wechat",
		"amount":       map[string]any{"total": pkg.AmountCents, "currency": "CNY"},
		"payer":        map[string]any{"openid": openid},
	})
	if err != nil {
		return nil, err
	}
	const path = "/v3/pay/transactions/jsapi"
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nc, err := nonce()
	if err != nil {
		return nil, err
	}
	sig, err := w.sign("POST\n" + path + "\n" + ts + "\n" + nc + "\n" + string(body) + "\n")
	if err != nil {
		return nil, err
	}
	base := w.PayURL
	if base == "" {
		base = "https://api.mch.weixin.qq.com"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf(`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",signature="%s",timestamp="%s",serial_no="%s"`, w.MchID, nc, sig, ts, w.MchSerial))
	resp, err := s.client().Do(req)
	if err != nil {
		return nil, errors.New("wechat pay: can't reach WeChat")
	}
	defer resp.Body.Close()
	var out struct {
		PrepayID string `json:"prepay_id"`
		Code     string `json:"code"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&out)
	if resp.StatusCode != http.StatusOK || out.PrepayID == "" {
		return nil, fmt.Errorf("wechat pay: status %d, code %q", resp.StatusCode, out.Code)
	}
	p := &PrepayParams{TimeStamp: strconv.FormatInt(time.Now().Unix(), 10), Package: "prepay_id=" + out.PrepayID, SignType: "RSA"}
	if p.NonceStr, err = nonce(); err != nil {
		return nil, err
	}
	if p.PaySign, err = w.sign(w.AppID + "\n" + p.TimeStamp + "\n" + p.NonceStr + "\n" + p.Package + "\n"); err != nil {
		return nil, err
	}
	return p, nil
}

// HandleWeChatNotify takes WeChat Pay's report of a payment: it checks the signature, opens
// the encrypted body, and adds the credits once.
func (s *Service) HandleWeChatNotify(ctx context.Context, payload []byte, h http.Header, now time.Time) (*Paid, error) {
	w := s.WeChat
	if w == nil {
		return nil, ErrBadSignature
	}
	ts := h.Get("Wechatpay-Timestamp")
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || h.Get("Wechatpay-Serial") != w.PublicKeyID {
		return nil, ErrBadSignature
	}
	if d := now.Sub(time.Unix(sec, 0)); d > signatureTolerance || d < -signatureTolerance {
		return nil, ErrBadSignature
	}
	sig, err := base64.StdEncoding.DecodeString(h.Get("Wechatpay-Signature"))
	if err != nil {
		return nil, ErrBadSignature
	}
	sum := sha256.Sum256([]byte(ts + "\n" + h.Get("Wechatpay-Nonce") + "\n" + string(payload) + "\n"))
	if rsa.VerifyPKCS1v15(w.PublicKey, crypto.SHA256, sum[:], sig) != nil {
		return nil, ErrBadSignature
	}
	var ev struct {
		EventType string `json:"event_type"`
		Resource  struct {
			Ciphertext     string `json:"ciphertext"`
			Nonce          string `json:"nonce"`
			AssociatedData string `json:"associated_data"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	if ev.EventType != "TRANSACTION.SUCCESS" {
		return nil, nil
	}
	raw, err := base64.StdEncoding.DecodeString(ev.Resource.Ciphertext)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher([]byte(w.APIv3Key))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, len(ev.Resource.Nonce))
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, []byte(ev.Resource.Nonce), raw, []byte(ev.Resource.AssociatedData))
	if err != nil {
		return nil, errors.New("wechat pay: can't open the notification")
	}
	var tx struct {
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		TradeState    string `json:"trade_state"`
		Amount        struct {
			Total    int    `json:"total"`
			Currency string `json:"currency"`
		} `json:"amount"`
	}
	if err := json.Unmarshal(plain, &tx); err != nil {
		return nil, err
	}
	if tx.TradeState != "SUCCESS" {
		return nil, nil
	}
	return s.completePurchase(ctx, tx.OutTradeNo, tx.TransactionID, tx.Amount.Total, tx.Amount.Currency)
}
