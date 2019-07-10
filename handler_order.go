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

	ErrorMessageNotifyDefaultError = "上下文校验失败：检查参数合法性"
)

var OrderIDCharCandidates = []byte("abcdefghijklmnopqrstuvwxyz0123456789")

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
	})
	if db.Error != nil {
		LogDb.Printf("find orders error: %v", db.Error)
		return NewErrorResponse(http.StatusBadRequest, "查找订单失败")
	}

	paginator := pagination.Paging(&pagination.Param{
		DB:      db,
		Page:    query.Page,
		Limit:   query.Limit,
		OrderBy: []string{"paid_at asc"},
		ShowSQL: true,
	}, &orders)
	return c.JSON(http.StatusOK, paginator)
}

func queryOrderStatus(c echo.Context) error {
	var order Order
	err := WebData.Where(&Order{
		OrderID:        c.Param("orderId"),
		ParentUsername: c.Get("token").(*Token).ParentUsername,
	}).Last(&order).Error
	if err != nil {
		LogDb.Printf("query order error: %v", err)
		return NewErrorResponse(http.StatusBadRequest, "未找到订单")
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

	// generate a 32 bytes-long containing only OrderIDCharCandidates characters random string as the new orderId
	orderId := uniuri.NewLenChars(32, OrderIDCharCandidates)

	// sends the payment request
	response, err := PaySession.Pay(xorpay.Transaction{
		Name:    "Life 币充值",
		PayType: form.Payment,
		Price:   form.Price,
		OrderID: orderId,
	})
	if err != nil {
		LogPay.Printf("create order error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "支付发起失败，稍后请重试")
	}

	err = WebData.Create(&Order{
		OrderID:         orderId,
		PlatformOrderID: response.PlatformOrderID,
		ParentUsername:  c.Get("token").(*Token).ParentUsername,
		PayType:         form.Payment,
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
		return NewErrorResponse(http.StatusNotAcceptable, ErrorMessageSignInvalid)
	}

	timezone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		LogPay.Printf("read timezone error: %v", err)
		return c.String(http.StatusInternalServerError, "服务器内部错误")
	}

	paidAt, err := time.ParseInLocation("2006-01-02 15:04:05", form.PayTime, timezone)
	if err != nil {
		LogPay.Printf("parse time error: %v", err)
		return DefaultBadRequestResponse
	}

	var detail xorpay.PlatformNotifyResponseDetail
	if err = json.Unmarshal([]byte(form.Detail), &detail); err != nil {
		LogPay.Printf("unmarshal error: %v", err)
		return DefaultBadRequestResponse
	}

	// search for the corresponding order
	db := WebData.Where(&Order{
		PlatformOrderID: form.PlatformOrderID,
		ParentUsername:  c.Get("token").(*Token).ParentUsername,
	})

	if err = db.Error; err != nil {
		LogPay.Printf("find initial order error: %v", err)
		return NewErrorResponse(http.StatusFailedDependency, "无对应用户订单记录")
	}

	var order Order
	if err = db.Find(&order).Error; err != nil {
		LogPay.Printf("find initial order error: %v", err)
		return NewErrorResponse(http.StatusInternalServerError, ErrorMessageDatabaseError)
	}

	// the order has already been saved before. abort to prevent saving duplicated order information.
	if order.PaidAt != nil {
		LogPay.Printf("attempt to save duplicated order %v with already existing order %v",
			spew.Sdump(form), spew.Sdump(order))
		return NewErrorResponse(http.StatusConflict, "重复的订单记录")
	}

	// according to form posted, update the corresponding order
	err = db.Update(&Order{
		PaidAt:          &paidAt,
		TransactionID:   detail.TransactionID,
		TransactionType: detail.TransactionType,
		//TransactionBuyer: detail.TransactionBuyer,
	}).Error
	if err != nil {
		LogPay.Printf("update order error: %v", err)
		return NewErrorResponse(http.StatusInternalServerError, ErrorMessageDatabaseError)
	}

	//err = GameData.Create(&PaidOrder{
	//	OrderID:   order.OrderID,
	//	Username:  c.Get("token").(*Token).ParentUsername,
	//	CreatedAt: order.CreatedAt,
	//	PaidAt:    paidAt,
	//	PaidPrice: order.PaidPrice,
	//	Processed: false,
	//}).Error
	//if err != nil {
	//	LogPay.Printf("store order to game db error: %v", err)
	//	return DefaultBadRequestResponse
	//}

	return c.NoContent(http.StatusAccepted)
}
