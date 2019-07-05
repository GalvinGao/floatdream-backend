package floatdream_backend_copy

import (
	"encoding/json"
	"github.com/labstack/echo"
	"net/http"
)

type ValidateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func validateUser(c echo.Context) error {
	var form EncryptedForm
	err := c.Bind(&form)
	if err != nil {
		return NewErrorResponse(http.StatusBadRequest, ErrorMessageInputParseError)
	}

	var request ValidateUserRequest
	obj, err := Decrypt.Decrypt(form)
	if err != nil {
		return err
	}
	err = json.Unmarshal(obj, &request)
	if err != nil {
		return err
	}

	
}

func invalidateUser(c echo.Context) error {

}
