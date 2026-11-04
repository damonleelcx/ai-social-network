// Package externalauth asks other services who someone is, for signing in with them.
package externalauth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ErrRejected means the service didn't accept what the person's device sent.
var ErrRejected = errors.New("the sign-in was rejected")

// WeChat turns the code from wx.login into the person's openid for this mini program.
type WeChat struct {
	AppID     string
	AppSecret string
	// URL is for tests. Empty means WeChat itself.
	URL  string
	HTTP *http.Client
}

func (w *WeChat) OpenID(ctx context.Context, code string) (string, error) {
	base := w.URL
	if base == "" {
		base = "https://api.weixin.qq.com"
	}
	q := url.Values{"appid": {w.AppID}, "secret": {w.AppSecret}, "js_code": {code}, "grant_type": {"authorization_code"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/sns/jscode2session?"+q.Encode(), nil)
	if err != nil {
		return "", errors.New("wechat login: bad request")
	}
	client := w.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		// This error quotes the URL, and the URL holds the app secret.
		return "", errors.New("wechat login: can't reach WeChat")
	}
	defer resp.Body.Close()
	var out struct {
		OpenID string `json:"openid"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&out); err != nil {
		return "", errors.New("wechat login: unreadable answer")
	}
	if out.OpenID == "" {
		return "", ErrRejected
	}
	return out.OpenID, nil
}
