package main

import (
	"database/sql"
	"time"
)

type DatabaseConfig struct {
	Source string `yaml:"source"`
	DSN    string `yaml:"dsn"`
}

type Config struct {
	Server struct {
		Address string `yaml:"address"`
		CORS    struct {
			Enabled      bool     `yaml:"enabled"`
			AllowOrigins []string `yaml:"allowOrigins"`
		} `yaml:"cors"`
	} `yaml:"server"`
	Database struct {
		Web    DatabaseConfig `yaml:"web"`
		AuthMe DatabaseConfig `yaml:"authMe"`
		Game   DatabaseConfig `yaml:"game"`
	} `yaml:"database"`
	Game struct {
		Address string `yaml:"address"`
	} `yaml:"game"`
	ReCAPTCHA struct {
		Secret string `yaml:"secret"`
	} `yaml:"recaptcha"`
	XorPay struct {
		AppID     string `yaml:"appId"`
		AppSecret string `yaml:"appSecret"`
		NotifyURL string `yaml:"notifyUrl"`
	}
}

// AuthMeUser describes a AuthMe user object
// Used when connecting to AuthMe database and validate user from there
type AuthMeUser struct {
	Username     string        `json:"username"`
	Password     string        `json:"password"`
	LastLoginRaw sql.NullInt64 `gorm:"column:authmelogin" json:"-"`
	LastLogin    time.Time     `gorm:"-" json:"last_login"`
	LoggedInRaw  uint          `gorm:"type:smallint(6);column:isLogged" json:"-"`
	LoggedIn     bool          `gorm:"-" json:"logged_in"`
}

func (u *AuthMeUser) TableName() string {
	return "authme"
}

func (u *AuthMeUser) AfterFind() (err error) {
	u.LoggedIn = u.LoggedInRaw == 0
	if u.LastLoginRaw.Valid {
		u.LastLogin = time.Unix(u.LastLoginRaw.Int64/1000, 0)
	}
	return
}

type Token struct {
	Token          string    `gorm:"char(32);primary_key" json:"token"`
	ExpireAt       time.Time `json:"expire_at"`
	ParentUsername string    `json:"parent_username"`
}

// PlatformOrder describes a order object in payment platform.
// Used when receives a callback from payment platform
//type PlatformOrder struct {
//	PlatformOrderID string `gorm:"size:32;unique_index" json:"-"`
//	ParentOrderID   string `gorm:"size:32" json:"-"`
//
//	PaidPrice int64     `json:"paid_price"`
//	PaidAt  time.Time `json:"paid_time"`
//
//	TransactionID    string `gorm:"unique_index" json:"transaction_id"`
//	TransactionType  string `json:"transaction_type"`
//	TransactionBuyer string `json:"transaction_buyer"`
//}

// Order describes a order object in this program
type Order struct {
	OrderID         string `gorm:"size:32;unique_index;NOT NULL" json:"order_id"`
	PlatformOrderID string `gorm:"size:32;unique_index;NOT NULL" json:"-"`
	ParentUsername  string `gorm:"size:255;index;NOT NULL" json:"-"`
	PayType         string `gorm:"size:32;NOT NULL" json:"pay_type"`

	CreatedAt time.Time `gorm:"NOT NULL" json:"created_at"`

	Paid      bool       `gorm:"-" json:"paid"`
	PaidPrice uint64     `gorm:"size:8;NOT NULL" json:"paid_price"`
	PaidAt    *time.Time `json:"paid_at"`

	TransactionID   string `gorm:"size:64" json:"-"`
	TransactionType string `gorm:"size:64" json:"transaction_type"`
	//TransactionBuyer string `json:"-"`

	Processed   bool       `gorm:"-" json:"processed"`
	ProcessedAt *time.Time `json:"processed_at"`
}

func (o *Order) AfterFind() (err error) {
	o.Paid = o.PaidAt != nil
	o.Processed = o.ProcessedAt != nil
	return
}

// PaidOrder describes an order which has been paid and is going to be stored in Game Database for topup purposes.
//type PaidOrder struct {
//	OrderID   string    `gorm:"size:32;unique_index" json:"order_id"`
//	Username  string    `gorm:"size:255;index" json:"username"`
//	CreatedAt time.Time `json:"created_at"`
//	PaidAt    time.Time `json:"paid_at"`
//	PaidPrice uint64    `json:"paid_price"`
//	Processed bool      `json:"processed"`
//}

type EncryptedForm struct {
	Payload string `json:"payload"`
}
