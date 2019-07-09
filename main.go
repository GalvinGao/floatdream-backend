package main

import (
	"github.com/GalvinGao/floatdream-backend/xorpay"
	rice "github.com/GeertJohan/go.rice"
	"github.com/davecgh/go-spew/spew"
	"github.com/jinzhu/configor"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/romanyx/recaptcha.v1"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	ErrorMessageBadRequest    = "请求参数错误"
	ErrorMessageDatabaseError = "数据库错误"
)

var (
	PaySession xorpay.Session

	WebData    *gorm.DB
	AuthMeData *gorm.DB
	GameData   *gorm.DB

	LogDb   *log.Logger
	LogPay  *log.Logger
	LogAuth *log.Logger

	ReCAPTCHAValidator *recaptcha.Client

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
		time.Sleep(time.Second)
		return c.JSON(http.StatusUnauthorized, ErrorMessage{
			"[simulation] 鉴权失败",
		})
	}
}

func main() {
	LogDb = log.New(os.Stdout, "Database", log.Ldate|log.Ltime|log.Lshortfile)
	LogPay = log.New(os.Stdout, "Payment", log.Ldate|log.Ltime|log.Lshortfile)
	LogAuth = log.New(os.Stdout, "Authorization", log.Ldate|log.Ltime|log.Lshortfile)

	// load configurations
	var config Config
	err := configor.Load(&config, "config.yml")
	if err != nil {
		LogDb.Panic("config file error", err)
	}

	spew.Dump(config)

	// initialize the payment api
	PaySession = xorpay.New(config.XorPay.NotifyURL, config.XorPay.AppID, config.XorPay.AppSecret)

	// initialize the ReCAPTCHA validator
	ReCAPTCHAValidator = recaptcha.New(config.ReCAPTCHA.Secret)

	// initialize database connection
	if WebData, err = gorm.Open(config.Database.Web.Source, config.Database.Web.DSN); err != nil {
		LogDb.Panic("failed to open database: `web`;", err)
	}

	// initialize database tables
	WebData.AutoMigrate(&Token{}, &Order{})

	if AuthMeData, err = gorm.Open(config.Database.AuthMe.Source, config.Database.AuthMe.DSN); err != nil {
		LogDb.Panic("failed to open database: `authme`;", err)
	}

	// check if the table exists or not
	if !AuthMeData.HasTable(&AuthMeUser{}) {
		LogDb.Panic("expect to see table `authme` in database `authme`")
	}

	if GameData, err = gorm.Open(config.Database.Game.Source, config.Database.Game.DSN); err != nil {
		LogDb.Panic("failed to open database: `game`;", err)
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

	assetHandler := http.FileServer(rice.MustFindBox("ui").HTTPBox())
	e.GET("/", echo.WrapHandler(assetHandler))
	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", assetHandler)))

	api := e.Group("/api")
	{
		api.GET("/status", fetchServerStatus(config.Game.Address))
		user := api.Group("/user")
		{
			user.POST("/login", validateUser)
			user.POST("/logout", invalidateUser, needValidation)
			user.GET("/info", retrieveUserInfo, needValidation)
			user.PATCH("/nickname", changeNickname, needValidation)
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

	e.GET("*", func(c echo.Context) error {
		file, err := rice.MustFindBox("ui").Bytes("index.html")
		if err != nil {
			return NewErrorResponse(http.StatusInternalServerError, "Handle 访问失败")
		}
		return c.Blob(http.StatusOK, "text/html", file)
	})

	log.Fatalln(e.Start(config.Server.Address))
}
