package main

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/labstack/echo"
	"net/http"
	"strings"
	"time"
)

const (
	ErrorMessageNeedAuthorization = "需要身份验证"
	ErrorMessageSessionExpired    = "用户会话已过期"
	ErrorMessageTokenSaveError    = "用户密钥延期失败"

	TokenLifetime = time.Hour * 24
)

func needValidation(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		credentials := c.Request().Header.Get("authorization")
		userToken := strings.TrimPrefix(credentials, "Bearer ")

		var token Token
		err := WebData.Where(&Token{
			Token: userToken,
		}).First(&token).Error
		if err != nil {
			LogAuth.Printf("validate token error: %v", err)
			return NewErrorResponse(http.StatusUnauthorized, ErrorMessageNeedAuthorization)
		}

		var tokens []Token
		err = WebData.Where(&Token{
			ParentUsername: token.ParentUsername,
		}).Offset(1).Find(&tokens).Error
		if err == nil {
			for _, v := range tokens {
				if v.ExpireAt.Before(time.Now()) {
					WebData.Delete(v)
				}
			}
		}

		if token.ExpireAt.Before(time.Now()) {
			WebData.Delete(token)
			LogAuth.Printf("token outdated for: %v", token)
			return NewErrorResponse(http.StatusUpgradeRequired, ErrorMessageSessionExpired)
		}

		token.ExpireAt = time.Now().Add(TokenLifetime)
		err = WebData.Save(&token).Error
		if err != nil {
			LogAuth.Printf("save token error: %v", err)
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
	return hex.EncodeToString(saltedPasswordString[:])
}

func checkUserCredentials(expectedPasswordHashOrigin string, attemptPassword string) bool {
	expectedPasswordSegments := strings.Split(expectedPasswordHashOrigin, "$")
	if expectedPasswordSegments[1] != "SHA" {
		return false
	}
	passwordSalt := expectedPasswordSegments[2]
	expectedPasswordHash := expectedPasswordSegments[3]

	attemptPasswordHash := authMeCalculateHash(attemptPassword, passwordSalt)

	return expectedPasswordHash == attemptPasswordHash
}
