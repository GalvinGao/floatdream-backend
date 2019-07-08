package main

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"net/http"
	"strings"
	"time"
)

const (
	ErrorMessageNeedAuthorization = "需要身份验证"
	ErrorMessageSessionExpired    = "用户会话已过期"
	ErrorMessageTokenSaveError    = "用户密钥延期失败"
)

func needValidation(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		log.Debug("validating credentials...")
		credentials := c.Request().Header.Get("authorization")
		userToken := strings.TrimPrefix(credentials, "Bearer ")

		var token Token
		err := WebData.First(&Token{
			Token: userToken,
		}).Find(&token).Error
		if err != nil {
			log.Debug("credentials validation failed...")
			spew.Dump(err)
			return NewErrorResponse(http.StatusUnauthorized, ErrorMessageNeedAuthorization)
		}

		if token.ExpireAt.Before(time.Now()) {
			WebData.Delete(token)
			return NewErrorResponse(http.StatusUpgradeRequired, ErrorMessageSessionExpired)
		}

		token.ExpireAt = time.Now().Add(time.Hour * 24)
		err = WebData.Save(&token).Error
		if err != nil {
			return NewErrorResponse(http.StatusUnauthorized, ErrorMessageTokenSaveError)
		}

		c.Set("token", &token)

		return next(c)
	}
}

func authMeCalculateHash(password string, salt string) string {
	hashedPasswordBytes := sha256.Sum256([]byte(password))
	hashedPasswordString := hex.EncodeToString(hashedPasswordBytes[:])
	saltedPasswordBytes := []byte(hashedPasswordString + salt)
	saltedPasswordString := sha256.Sum256(saltedPasswordBytes)
	spew.Dump(hashedPasswordBytes, hashedPasswordString, saltedPasswordBytes, saltedPasswordString)
	return hex.EncodeToString(saltedPasswordString[:])
}

func checkUserCredentials(expectedPasswordHashOrigin string, attemptPassword string) bool {
	expectedPasswordSegments := strings.Split(expectedPasswordHashOrigin, "$")
	if expectedPasswordSegments[1] != "SHA" {
		return false
	}
	passwordSalt := expectedPasswordSegments[2]
	expectedPasswordHash := expectedPasswordSegments[3]
	spew.Dump(passwordSalt, expectedPasswordHash)

	attemptPasswordHash := authMeCalculateHash(attemptPassword, passwordSalt)
	spew.Dump(attemptPasswordHash)

	return expectedPasswordHash == attemptPasswordHash
}
