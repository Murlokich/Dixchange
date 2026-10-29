package http

import (
	"net/http"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	cancelorder "github.com/Murlokich/Dixchange/internal/domain/usecase/cancel_order"
	createorder "github.com/Murlokich/Dixchange/internal/domain/usecase/create_order"
	"github.com/Murlokich/Dixchange/internal/domain/usecase/deposit"
	getorders "github.com/Murlokich/Dixchange/internal/domain/usecase/get_orders"
	getpairbalance "github.com/Murlokich/Dixchange/internal/domain/usecase/get_pair_balance"
	gettrades "github.com/Murlokich/Dixchange/internal/domain/usecase/get_trades"
	"github.com/Murlokich/Dixchange/internal/domain/usecase/login"
	"github.com/Murlokich/Dixchange/internal/domain/usecase/register"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/pkg/errors"
)

const (
	invalidRequestParametersResponse       = "Invalid request parameters"
	invalidCredentialsResponse             = "Invalid credentials" //nolint:gosec // False positive for hardcoded creds
	registrationFailureResponse            = "Registration failed due to internal error"
	depositFailureResponse                 = "Deposit failed due to internal error"
	loginFailureResponse                   = "Login failed due to internal error"
	createOrderFailureResponse             = "Order creation failed due to internal error"
	createOrderInsufficientBalanceResponse = "Order creation faild due to insufficient balance"
	cancelOrderFailureResponse             = "Order cancellation failed due to internal error"
	cancelOrderNotCancellableResponse      = "Order is no longer active and can't be cancelled"
	getMarketDataFailureResponse           = "Market data retrieval failed due to internal error"
	getPairBalanceFailureResponse          = "Pair balance retrieval failed due to internal error"
	getPairBalanceNotFoundResponse         = "Balance not found"
	getOrdersFailureResponse               = "Orders retrieval failed due to internal error"
	getTradesFailureResponse               = "Trades retrieval failed due to internal error"

	errorField = "error"

	userIDParamKey = "user_id"
	loginOKField   = "ok"
)

func loginHandler(c *gin.Context, deps *Dependencies) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.Logger.Error(err.Error())
		return
	}

	output, err := deps.Login.Execute(c, login.Input{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.Logger.Error(err.Error())
			return
		}

		if errors.Is(err, login.ErrInvalidCredentials) {
			c.JSON(http.StatusForbidden, gin.H{errorField: invalidCredentialsResponse})
			deps.Logger.Error(err.Error())
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{errorField: loginFailureResponse})
		deps.Logger.Error(err.Error())
		return
	}
	setAuthCookies(c, output.AccessToken)
	c.JSON(http.StatusOK, gin.H{loginOKField: true})
}

func registerHandler(c *gin.Context, deps *Dependencies) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.Logger.Error(err.Error())
		return
	}
	_, err := deps.Register.Execute(c, register.Input{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.Logger.Error(err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{errorField: registrationFailureResponse})
		deps.Logger.Error(err.Error())
		return
	}

	c.Status(http.StatusCreated)
}

func depositHandler(c *gin.Context, deps *Dependencies) {
	var req DepositRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.Logger.Error(err.Error())
		return
	}

	userUD, err := GetUserIDFromCtx(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
		deps.Logger.Errorf("get user id from ctx: %v", err)
		return
	}

	_, err = deps.Deposit.Execute(c, deposit.Input{
		UserID:   userUD,
		SymbolID: req.SymbolID,
		Amount:   req.Amount,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.Logger.Error(err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{errorField: depositFailureResponse})
		deps.Logger.Error(err.Error())
		return
	}
	c.Status(http.StatusOK)
}

func createOrderHandler(c *gin.Context, deps *Dependencies) {
	var req CreateOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.Logger.Error(err.Error())
		return
	}
	userUD, err := GetUserIDFromCtx(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
		deps.Logger.Errorf("get user id from ctx: %v", err)
		return
	}

	_, err = deps.CreateOrder.Execute(c, createorder.Input{
		UserID:          userUD,
		MarketPairID:    req.MarketPairID,
		Side:            req.Side,
		QuantityOrdered: req.QuantityOrdered,
		PricePerUnit:    req.PricePerUnit,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.Logger.Error(err.Error())
			return
		}
		if errors.Is(err, entity.ErrInsufficientBalance) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: createOrderInsufficientBalanceResponse})
			deps.Logger.Error(err.Error())
			return
		}
		if errors.Is(err, entity.ErrCommandBusOverloaded) {
			c.Status(http.StatusServiceUnavailable)
			deps.Logger.Warnf("%v", err)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{errorField: createOrderFailureResponse})
		deps.Logger.Error(err.Error())
		return
	}
	c.Status(http.StatusCreated)
}

// cancelOrderHandler answers 202 once the cancel request is handed to the matching
// engine: the engine decides whether the order is still there to cancel.
//
// A cancel of an order that doesn't exist or belongs to another user gets the same 202,
// so the requester learns nothing about other users' orders or that the attempt was
// noticed. The request goes no further, it is only logged.
func cancelOrderHandler(c *gin.Context, deps *Dependencies) {
	var req CancelOrderRequest

	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.Logger.Error(err.Error())
		return
	}
	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
		deps.Logger.Errorf("parse order id: %v", err)
		return
	}

	userID, err := GetUserIDFromCtx(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
		deps.Logger.Errorf("get user id from ctx: %v", err)
		return
	}

	_, err = deps.CancelOrder.Execute(c, cancelorder.Input{
		UserID:  userID,
		OrderID: orderID,
	})
	if err != nil {
		switch {
		case errors.As(err, &entity.ValidationError{}):
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.Logger.Error(err.Error())
		case errors.Is(err, entity.ErrOrderNotOwned):
			c.Status(http.StatusAccepted)
			deps.Logger.Warnf("order ownership mismatch on cancel: %v", err)
		case errors.Is(err, entity.ErrOrderNotFound):
			c.Status(http.StatusAccepted)
			deps.Logger.Infof("cancel of order that doesn't exist: %v", err)
		case errors.Is(err, entity.ErrOrderNotCancellable):
			c.JSON(http.StatusConflict, gin.H{errorField: cancelOrderNotCancellableResponse})
			deps.Logger.Error(err.Error())
		case errors.Is(err, entity.ErrCommandBusOverloaded):
			c.Status(http.StatusServiceUnavailable)
			deps.Logger.Warnf("%v", err)
		default:
			c.JSON(http.StatusInternalServerError, gin.H{errorField: cancelOrderFailureResponse})
			deps.Logger.Error(err.Error())
		}
		return
	}
	c.Status(http.StatusAccepted)
}

func getMarketDataHandler(c *gin.Context, deps *Dependencies) {
	output, err := deps.GetMarketData.Execute(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{errorField: getMarketDataFailureResponse})
		deps.Logger.Error(err.Error())
		return
	}
	c.JSON(http.StatusOK, newMarketDataResponse(output.Pairs))
}

func getPairBalanceHandler(c *gin.Context, deps *Dependencies) {
	var req GetPairBalanceRequest

	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.Logger.Error(err.Error())
		return
	}

	userID, err := GetUserIDFromCtx(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
		deps.Logger.Errorf("get user id from ctx: %v", err)
		return
	}

	output, err := deps.GetPairBalance.Execute(c, getpairbalance.Input{
		UserID: userID,
		PairID: req.PairID,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.Logger.Error(err.Error())
			return
		}
		if errors.Is(err, entity.ErrBalanceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{errorField: getPairBalanceNotFoundResponse})
			deps.Logger.Error(err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{errorField: getPairBalanceFailureResponse})
		deps.Logger.Error(err.Error())
		return
	}
	c.JSON(http.StatusOK, newPairBalanceResponse(output.Pair, output.Base, output.Quote))
}

func getOrdersHandler(c *gin.Context, deps *Dependencies) {
	var req GetOrdersRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.Logger.Error(err.Error())
		return
	}

	userID, err := GetUserIDFromCtx(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
		deps.Logger.Errorf("get user id from ctx: %v", err)
		return
	}

	output, err := deps.GetOrders.Execute(c, getorders.Input{
		UserID:   userID,
		Statuses: req.Statuses,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.Logger.Error(err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{errorField: getOrdersFailureResponse})
		deps.Logger.Error(err.Error())
		return
	}
	c.JSON(http.StatusOK, newOrdersResponse(output.Orders))
}

func getTradesHandler(c *gin.Context, deps *Dependencies) {
	var req GetTradesRequest

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.Logger.Error(err.Error())
		return
	}

	var cursor *uuid.UUID
	if req.Cursor != "" {
		parsed, err := uuid.Parse(req.Cursor)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.Logger.Errorf("parse trade cursor: %v", err)
			return
		}
		cursor = &parsed
	}

	userID, err := GetUserIDFromCtx(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
		deps.Logger.Errorf("get user id from ctx: %v", err)
		return
	}

	output, err := deps.GetTrades.Execute(c, gettrades.Input{
		UserID: userID,
		Cursor: cursor,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.Logger.Error(err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{errorField: getTradesFailureResponse})
		deps.Logger.Error(err.Error())
		return
	}
	c.JSON(http.StatusOK, newTradesResponse(output))
}
