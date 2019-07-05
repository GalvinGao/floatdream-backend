package floatdream_backend_copy

import (
	"database/sql"
	"time"
)

type Config struct {
	Server struct {
		Address string `yaml:"address"`
		CORS bool `yaml:"cors"`
	} `yaml:"server"`
	Database struct {
		Web string `yaml:"web"`
		AuthMe string `yaml:"authMe"`
		Game string `yaml:"game"`
	} `yaml:"database"`
	Encrypt struct {
		PrivateKey string `yaml:"privateKey"`
	} `yaml:"encrypt"`
}

// AuthMeUser describes a AuthMe user object
// Used when connecting to AuthMe database and validate user from there
type AuthMeUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
	LastLoginRaw sql.NullInt64 `gorm:"column:authmelogin" json:"-"`
	LastLogin time.Time `gorm:"-" json:"last_login"`
	LoggedInRaw uint `gorm:"type:smallint(6)" json:"-"`
	LoggedIn bool `gorm:"-" json:"logged_in"`

	Tokens []Token `gorm:"foreignkey:ParentUsername" json:"tokens"`
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
	TokenID int `gorm:"primary_key" json:"token_id"`
	ParentUsername string `json:"parent_username"`
	Token string `gorm:"char(32)" json:"token"`
}

// PlatformOrder describes a order object in payment platform.
// Used when receives a callback from payment platform
type PlatformOrder struct {
	PlatformOrderID string `gorm:"unique_index" json:"platform_order_id"`

	PaidPrice int64 `json:"paid_price"`
	PaidTime time.Time `json:"paid_time"`

	TransactionID string `gorm:"unique_index" json:"transaction_id"`
	TransactionType string `json:"transaction_type"`
	TransactionBuyer string `json:"transaction_buyer"`
}

// Order describes a order object in this program
type Order struct {
	OrderID uint64 `gorm:"unique_index" json:"order_id"`
	PlatformOrder PlatformOrder `gorm:"foreignkey:PlatformOrderID" json:"platform_order"`

	ParentUsername string `gorm:"type:varchar(255);index" json:"parent_username"`

	CreatedAt time.Time `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`

	PaidAmount sql.NullInt64 `json:"paid_amount"`
}

type EncryptedForm struct {
	Payload string `json:"payload"`
}

