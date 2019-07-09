package main

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/dchest/uniuri"
	"github.com/labstack/echo"
	"net/http"
	"time"
)

const (
	ErrorMessageNoSuchUser    = "无此用户"
	ErrorMessagePasswordError = "密码错误"
	ErrorMessageTokenError    = "内部错误 0x01"
)

type UserLoginRequest struct {
	Username  string `json:"username" validate:"required"`
	Password  string `json:"password" validate:"required"`
	ReCAPTCHA string `json:"recaptcha" validate:"required"`
}

type UserLoginResponse struct {
	Username string `json:"username"`
	Token    string `json:"token"`
	Nickname string `json:"nickname"`
}

type UserInfoResponse struct {
	StatusOnline     bool       `json:"status_online"`
	StatusLastLogin  time.Time  `json:"status_last_login"`
	BalanceCurrent   uint64     `json:"balance_current"`
	BalanceLastTopup *time.Time `json:"balance_last_topup,emitempty"`
}

func validateUser(c echo.Context) error {
	var form UserLoginRequest
	if err := c.Bind(&form); err != nil {
		LogAuth.Printf("bind form error: %v", err)
		return DefaultBadRequestResponse
	}
	if err := c.Validate(&form); err != nil {
		LogAuth.Printf("validate form error: %v", err)
		return DefaultBadRequestResponse
	}

	_, err := ReCAPTCHAValidator.VerifyWithIP(form.ReCAPTCHA, c.RealIP())
	if err != nil {
		LogAuth.Printf("recaptcha error: %v", err)
		return NewErrorResponse(http.StatusBadRequest, "reCAPTCHA verification failed")
	}

	spew.Dump(form)

	var attemptValidateUser AuthMeUser
	err = AuthMeData.Where(&AuthMeUser{
		Username: form.Username,
	}).Find(&attemptValidateUser).Error
	if err != nil {
		LogDb.Printf("find user error: %v", err)
		return NewErrorResponse(http.StatusBadRequest, ErrorMessageNoSuchUser)
	}

	ok := checkUserCredentials(attemptValidateUser.Password, form.Password)
	if !ok {
		return NewErrorResponse(http.StatusBadRequest, ErrorMessagePasswordError)
	}

	token := uniuri.NewLen(32)
	err = WebData.Create(&Token{
		Token:          token,
		ExpireAt:       time.Now().Add(TokenLifetime),
		ParentUsername: attemptValidateUser.Username,
	}).Error
	if err != nil {
		LogDb.Printf("create token error: %v", err)
		return NewErrorResponse(http.StatusBadRequest, ErrorMessageTokenError)
	}

	return c.JSON(http.StatusAccepted, UserLoginResponse{
		Username: attemptValidateUser.Username,
		Token:    token,
		Nickname: "", // TODO: contact acid to retrieve nickname information.
	})
}

func invalidateUser(c echo.Context) error {
	if err := WebData.Delete(c.Get("token").(*Token)).Error; err != nil {
		LogDb.Printf("delete token error: %v", err)
		return NewErrorResponse(http.StatusBadRequest, ErrorMessageTokenError)
	}
	return c.NoContent(http.StatusOK)
}

func retrieveUserInfo(c echo.Context) error {
	username := c.Get("token").(*Token).ParentUsername
	var user AuthMeUser
	err := AuthMeData.First(&AuthMeUser{
		Username: username,
	}).Find(&user).Error
	if err != nil {
		LogDb.Printf("find user error: %v", err)
		return NewErrorResponse(http.StatusBadRequest, ErrorMessageDatabaseError)
	}

	var paidTime *time.Time

	var latestOrder Order
	err = WebData.Order("paid_time DESC").Last(&Order{
		ParentUsername: username,
	}).Find(&latestOrder).Error
	if err != nil {
		LogDb.Printf("get latest order error: %v", err)
		return NewErrorResponse(http.StatusBadRequest, ErrorMessageDatabaseError)
	}
	if latestOrder.PaidTime != nil {
		paidTime = nil
	} else {
		paidTime = latestOrder.PaidTime
	}

	return c.JSON(http.StatusOK, UserInfoResponse{
		StatusOnline:     user.LoggedIn,
		StatusLastLogin:  user.LastLogin,
		BalanceCurrent:   120, // TODO: contact acid to ask for balance interface
		BalanceLastTopup: paidTime,
	})
}

func changeNickname(c echo.Context) error {
	return c.NoContent(http.StatusAccepted)
}
