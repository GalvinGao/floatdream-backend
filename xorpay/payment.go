package xorpay

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	PayURLTemplate = "https://xorpay.com/api/pay/%s"
	Timeout = time.Minute
)

type Session struct {
	NotifyURL string `json:"notify_url"`
	AppID string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	PayURL string `json:"pay_url"`

	Client http.Client `json:"-"`
}

type Transaction struct {
	Name string `json:"name"`
	PayType string `json:"pay_type"`
	Price uint64 `json:"price"`
	OrderID uint64 `json:"order_id"`
}

type PlatformPayResponse struct {
	Info struct {
		QR string `json:"qr"`
	} `json:"info"`
}

func New(notifyUrl string, appId string, appSecret string) Session {
	return Session{
		NotifyURL: notifyUrl,
		AppID:     appId,
		AppSecret: appSecret,
		PayURL:    fmt.Sprintf(PayURLTemplate, appId),
		Client: http.Client{
			Timeout: Timeout,
		},
	}
}

func calculateSign(t Transaction, s Session) string {
	params := fmt.Sprintf("%s%s%v%v%s%s", t.Name, t.PayType, t.Price, t.OrderID, s.NotifyURL, s.AppSecret)
	hash := md5.Sum([]byte(params))
	return hex.EncodeToString(hash[:])
}

func (s Session) Pay(t Transaction) (string, error) {
	v := url.Values{}
	v.Set("name", t.Name)
	v.Set("pay_type", t.PayType)
	v.Set("price", strconv.FormatUint(t.Price, 64))
	v.Set("order_id", strconv.FormatUint(t.OrderID, 64))
	v.Set("notify_url", s.NotifyURL)
	v.Set("sign", calculateSign(t, s))

	resp, err := http.PostForm(s.PayURL, v)
	if err != nil {
		return "", err
	}

	var platformPayResponse PlatformPayResponse
	err = json.NewDecoder(resp.Body).Decode(&platformPayResponse)
	if err != nil {
		return "", err
	}

	return platformPayResponse.Info.QR, nil
}




