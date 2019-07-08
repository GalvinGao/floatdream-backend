package main

import (
	"github.com/GalvinGao/floatdream-backend/xorpay"
	"github.com/davecgh/go-spew/spew"
	"github.com/jinzhu/configor"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"gopkg.in/go-playground/validator.v9"
	"log"
	"net/http"
	"time"
)

const (
	ErrorMessageBadRequest    = "请求参数错误"
	ErrorMessageDatabaseError = "数据库错误"
)

var (
	Decrypt    Decryptor
	PaySession xorpay.Session

	WebData    *gorm.DB
	AuthMeData *gorm.DB
	GameData   *gorm.DB

	DefaultBadRequestResponse = NewErrorResponse(http.StatusBadRequest, ErrorMessageBadRequest)
)

type Validator struct {
	validator *validator.Validate
}

func (cv *Validator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func SimulateErrorResponse(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		time.Sleep(time.Millisecond * 4000)
		return c.JSON(http.StatusBadRequest, ErrorMessage{
			"[simulation] 请求参数错误",
		})
	}
}

func main() {
	// load configurations
	var config Config
	err := configor.Load(&config, "config.yml")
	if err != nil {
		log.Panic("config file error", err)
	}

	spew.Dump(config)

	// get the decryptor
	Decrypt = NewDecryptor(config.Encryption.PrivateKey)

	// initialize the payment api
	PaySession = xorpay.New(config.XorPay.NotifyURL, config.XorPay.AppID, config.XorPay.AppSecret)

	// initialize database connection
	if WebData, err = gorm.Open(config.Database.Web.Source, config.Database.Web.DSN); err != nil {
		log.Panic("failed to open database: `web`;", err)
	}

	// initialize database tables
	WebData.AutoMigrate(&Token{}, &Order{})

	if AuthMeData, err = gorm.Open(config.Database.AuthMe.Source, config.Database.AuthMe.DSN); err != nil {
		log.Panic("failed to open database: `authme`;", err)
	}

	// check if the table exists or not
	if !AuthMeData.HasTable(&AuthMeUser{}) {
		log.Panic("expect to see table `authme` in database `authme`")
	}

	if GameData, err = gorm.Open(config.Database.Game.Source, config.Database.Game.DSN); err != nil {
		log.Panic("failed to open database: `game`;", err)
	}

	e := echo.New()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339} | ${status} ${method} ${uri} ${latency_human}\n",
	}))
	if config.Server.CORS.Enabled {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: config.Server.CORS.AllowOrigins,
			AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		}))
	}

	e.Validator = &Validator{
		validator: validator.New(),
	}

	api := e.Group("/api")
	{
		user := api.Group("/user")
		{
			user.POST("/login", validateUser)
			user.POST("/logout", invalidateUser, needValidation)
			user.GET("/info", retrieveUserInfo, needValidation)
			user.GET("/status", func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			}, needValidation)
		}

		topup := api.Group("/topup")
		{
			topup.GET("/item", itemDetails)
			order := topup.Group("/order", needValidation)
			{
				order.GET("", listOrder)
				order.GET("/:orderId/status", queryOrderStatus)
				order.POST("", placeOrder)
				order.POST("/callback", storeOrder)
			}
		}
	}

	log.Fatalln(e.Start(config.Server.Address))
}
