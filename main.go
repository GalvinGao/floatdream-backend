package floatdream_backend_copy

import (
	"github.com/jinzhu/configor"
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo"
	"log"
)

const (
	ErrorMessageInputParseError = "参数解析失败"
)

var (
	Decrypt     Decryptor
	WebData *gorm.DB
	AuthMeData    *gorm.DB
	GameData      *gorm.DB
)

func main() {
	// load configurations
	var config Config
	err := configor.Load(&config)
	if err != nil {
		log.Panic("config file error", err)
	}

	// get the decryptor
	Decrypt = NewDecryptor(config.Encrypt.PrivateKey)

	// initialize database connection
	if WebData, err = gorm.Open(config.Database.Web); err != nil {
		log.Panic("failed to open database: `web`;", err)
	}

	// initialize database tables
	WebData.AutoMigrate(&Token{}, &Order{}, &PlatformOrder{})

	if AuthMeData, err = gorm.Open(config.Database.AuthMe); err != nil {
		log.Panic("failed to open database: `authme`;", err)
	}

	// check if the table exists or not
	if !AuthMeData.HasTable(&AuthMeUser{}) {
		log.Panic("expect to see table `authme` in database `authme`")
	}

	if GameData, err = gorm.Open(config.Database.Game); err != nil {
		log.Panic("failed to open database: `game`;", err)
	}

	e := echo.New()

	user := e.Group("/user")
	{
		user.POST("/login", validateUser)
		user.POST("/logout", invalidateUser)
	}

	sponsor := e.Group("/sponsor")
	{
		sponsor.POST("/order", )
	}

	panic(e.Start(config.Server.Address))
}
