package main

import (
	"encoding/json"
	"github.com/GalvinGao/floatdream-backend/xorpay"
	"github.com/biezhi/gorm-paginator/pagination"
	"github.com/davecgh/go-spew/spew"
	"github.com/dchest/uniuri"
	"github.com/labstack/echo"
	"net/http"
	"time"
)

const (
	ErrorMessageGameBackendUnavailable = "游戏服务器无响应"
	ErrorMessageGameBackendBadRequest  = "游戏服务器返回了无效信息"
	ErrorMessageSignInvalid            = "Sign 校验失败"
)

type GameBackendItemRequest struct {
	Ratio uint `json:"ratio,string" xml:"ratio"`
}

type ItemDetailsResponse struct {
	Ratio uint `json:"ratio"`
}

type PaginationRequest struct {
	Page  int `json:"page,string" query:"page"`
	Limit int `json:"limit,string" query:"limit"`
}

type PlaceOrderRequest struct {
	Price   uint64 `json:"price,string" validate:"required"`
	Payment string `json:"payment" validate:"required,oneof=alipay native"`
}

type PlaceOrderResponse struct {
	OrderID   string `json:"orderId"`
	QRContent string `json:"qrContent"`
	ExpiresIn uint   `json:"expiresIn"`
}

func itemDetails(c echo.Context) error {
	//client := http.Client{
	//	Timeout: time.Second * 15,
	//}
	//
	//resp, err := client.Get("http://10.6.6.2/order/item/details")
	//if err != nil {
	//	return NewErrorResponse(http.StatusServiceUnavailable, ErrorMessageGameBackendUnavailable)
	//}
	//
	//var response GameBackendItemRequest
	//err = json.NewDecoder(resp.Body).Decode(&response)
	//if err != nil {
	//	return NewErrorResponse(http.StatusServiceUnavailable, ErrorMessageGameBackendBadRequest)
	//}

	return c.JSON(http.StatusOK, ItemDetailsResponse{
		Ratio: 1,
	})
}

func listOrder(c echo.Context) error {
	var query PaginationRequest
	if err := c.Bind(&query); err != nil {
		LogDb.Printf("list query error: %v", err)
		return DefaultBadRequestResponse
	}

	var orders []Order
	db := WebData.Where(&Order{
		ParentUsername: c.Get("token").(*Token).ParentUsername,
	}).Find(&orders)
	if db.Error != nil {
		LogDb.Printf("find orders error: %v", db.Error)
		return NewErrorResponse(http.StatusServiceUnavailable, ErrorMessageDatabaseError)
	}

	paginator := pagination.Paging(&pagination.Param{
		DB:      db,
		Page:    query.Page,
		Limit:   query.Limit,
		OrderBy: []string{"id desc"},
		ShowSQL: true,
	}, &orders)
	return c.JSON(http.StatusOK, paginator)
}

func queryOrderStatus(c echo.Context) error {
	var order Order
	err := WebData.Where(&Order{
		OrderID:        c.Param("orderId"),
		ParentUsername: c.Get("token").(*Token).ParentUsername,
	}).Find(&order).Error
	if err != nil {
		LogDb.Printf("query order error: %v", err)
		return NewErrorResponse(http.StatusBadRequest, ErrorMessageDatabaseError)
	}

	return c.JSON(http.StatusOK, order)
}

func placeOrder(c echo.Context) error {
	var form PlaceOrderRequest
	if err := c.Bind(&form); err != nil {
		LogPay.Printf("bind form error: %v", err)
		return DefaultBadRequestResponse
	}
	if err := c.Validate(&form); err != nil {
		LogPay.Printf("validate form error: %v", err)
		return DefaultBadRequestResponse
	}

	orderId := uniuri.NewLen(32)

	response, err := PaySession.Pay(xorpay.Transaction{
		Name:    "Life 币充值",
		PayType: form.Payment,
		Price:   form.Price,
		OrderID: orderId,
	})
	if err != nil {
		LogPay.Printf("create order error: %v", err)
		return DefaultBadRequestResponse
	}

	err = WebData.Create(&Order{
		OrderID:         orderId,
		PlatformOrderID: response.PlatformOrderID,
		ParentUsername:  c.Get("token").(*Token).ParentUsername,
		CreatedAt:       time.Now(),
		PaidPrice:       form.Price,
	}).Error
	if err != nil {
		LogDb.Printf("save order error: %v", err)
		return NewErrorResponse(http.StatusInternalServerError, ErrorMessageDatabaseError)
	}

	return c.JSON(http.StatusCreated, PlaceOrderResponse{
		OrderID:   orderId,
		QRContent: response.Info.QR,
		ExpiresIn: response.ExpiresIn,
	})
}

func storeOrder(c echo.Context) error {
	var form xorpay.PlatformNotifyResponse
	if err := c.Bind(&form); err != nil {
		LogPay.Printf("bind form error: %v", err)
		return DefaultBadRequestResponse
	}
	if err := c.Validate(&form); err != nil {
		LogPay.Printf("validate form error: %v", err)
		return DefaultBadRequestResponse
	}

	if !PaySession.CheckSign(&form) {
		LogPay.Printf("check sign error for form: %v", spew.Sdump(form))
		return NewErrorResponse(http.StatusUnauthorized, ErrorMessageSignInvalid)
	}

	timezone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		LogPay.Printf("read timezone error: %v", err)
		return c.String(http.StatusInternalServerError, "服务器内部错误")
	}

	paidTime, err := time.ParseInLocation("2006-01-02 15:04:05", form.PayTime, timezone)
	if err != nil {
		LogPay.Printf("parse time error: %v", err)
		return DefaultBadRequestResponse
	}

	var detail xorpay.PlatformNotifyResponseDetail
	if err = json.Unmarshal([]byte(form.Detail), &detail); err != nil {
		LogPay.Printf("unmarshal error: %v", err)
		return DefaultBadRequestResponse
	}

	err = WebData.Table("Orders").Where(&Order{
		PlatformOrderID: form.PlatformOrderID,
		ParentUsername:  c.Get("token").(*Token).ParentUsername,
	}).Update(&Order{
		PaidTime:         &paidTime,
		TransactionID:    detail.TransactionID,
		TransactionType:  detail.TransactionType,
		TransactionBuyer: detail.TransactionBuyer,
	}).Error
	if err != nil {
		LogPay.Printf("update order error: %v", err)
		return DefaultBadRequestResponse
	}

	return c.NoContent(http.StatusAccepted)
}
