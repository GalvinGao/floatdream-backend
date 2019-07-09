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
	if u.LoggedInRaw == 0 {
		u.LoggedIn = false
	} else {
		u.LoggedIn = true
	}

	if u.LastLoginRaw.Valid {
		u.LastLogin = time.Unix(u.LastLoginRaw.Int64, 0)
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
//	PaidTime  time.Time `json:"paid_time"`
//
//	TransactionID    string `gorm:"unique_index" json:"transaction_id"`
//	TransactionType  string `json:"transaction_type"`
//	TransactionBuyer string `json:"transaction_buyer"`
//}

// Order describes a order object in this program
type Order struct {
	OrderID         string `gorm:"size:32;unique_index" json:"order_id"`
	PlatformOrderID string `gorm:"size:32;unique_index" json:"-"`
	ParentUsername  string `gorm:"size:255;index" json:"-"`

	CreatedAt time.Time `json:"created_at"`

	PaidPrice uint64     `json:"paid_price"`
	PaidTime  *time.Time `json:"paid_time"`

	TransactionID    string `json:"-"`
	TransactionType  string `json:"transaction_type"`
	TransactionBuyer string `json:"-"`
}

type EncryptedForm struct {
	Payload string `json:"payload"`
}
