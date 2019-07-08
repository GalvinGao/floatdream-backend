package xorpay

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	PayURLTemplate = "https://xorpay.com/api/pay/%s"
	Timeout        = time.Minute
)

type Session struct {
	NotifyURL string `json:"notify_url"`
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
	PayURL    string `json:"pay_url"`

	Client http.Client `json:"-"`
}

type Transaction struct {
	Name    string `json:"name"`
	PayType string `json:"pay_type"`
	Price   uint64 `json:"price"`
	OrderID string `json:"order_id"`
}

type PlatformPayResponse struct {
	Status          string `json:"status"`
	ExpiresIn       uint   `json:"expires_in"`
	PlatformOrderID string `json:"aoid"`
	Info            struct {
		QR string `json:"qr"`
	} `json:"info"`
}

type PlatformNotifyResponse struct {
	PlatformOrderID string `json:"aoid" form:"aoid"`
	OrderID         string `json:"order_id" form:"order_id"`
	PayPrice        string `json:"pay_price" form:"pay_price"`
	PayTime         string `json:"pay_time" form:"pay_time"`
	Sign            string `json:"sign" form:"sign"`
	Detail          string `json:"detail" form:"detail"`
}

type PlatformNotifyResponseDetail struct {
	TransactionID    string `json:"transaction_id"`
	TransactionType  string `json:"bank_type"`
	TransactionBuyer string `json:"buyer"`
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
	concatenated := strings.Join([]string{
		t.Name,
		t.PayType,
		strconv.FormatUint(t.Price, 10),
		t.OrderID,
		s.NotifyURL,
		s.AppSecret,
	}, "")
	hash := md5.Sum([]byte(concatenated))
	return hex.EncodeToString(hash[:])
}

// Pay sends order info to xorpay
// returns payment response, error
func (s Session) Pay(t Transaction) (payResponse *PlatformPayResponse, err error) {
	v := url.Values{}
	v.Set("name", t.Name)
	v.Set("pay_type", t.PayType)
	v.Set("price", strconv.FormatUint(t.Price, 10))
	v.Set("order_id", t.OrderID)
	v.Set("notify_url", s.NotifyURL)
	v.Set("sign", calculateSign(t, s))

	resp, err := http.PostForm(s.PayURL, v)
	if err != nil {
		return &PlatformPayResponse{}, err
	}

	var platformPayResponse PlatformPayResponse
	err = json.NewDecoder(resp.Body).Decode(&platformPayResponse)
	if err != nil {
		return &PlatformPayResponse{}, err
	}
	if platformPayResponse.Status != "ok" {
		return &PlatformPayResponse{}, errors.New(fmt.Sprintf("platform response not ok: %s", platformPayResponse.Status))
	}

	return &platformPayResponse, nil
}

func (s Session) CheckSign(r *PlatformNotifyResponse) bool {
	concatenated := strings.Join([]string{
		r.PlatformOrderID,
		r.OrderID,
		r.PayPrice,
		r.PayTime,
		s.AppSecret,
	}, "")
	sumBytes := md5.Sum([]byte(concatenated))
	sumHex := hex.EncodeToString(sumBytes[:])
	spew.Dump(r, concatenated, sumBytes, sumHex)
	return sumHex == r.Sign
}
