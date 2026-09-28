package http

import (
	"net/http"

	"github.com/dixchange/Monolith/internal/domain/entity"
	createorder "github.com/dixchange/Monolith/internal/domain/usecase/create_order"
	"github.com/dixchange/Monolith/internal/domain/usecase/deposit"
	getorders "github.com/dixchange/Monolith/internal/domain/usecase/get_orders"
	getpairbalance "github.com/dixchange/Monolith/internal/domain/usecase/get_pair_balance"
	"github.com/dixchange/Monolith/internal/domain/usecase/login"
	"github.com/dixchange/Monolith/internal/domain/usecase/register"
	"github.com/gin-gonic/gin"

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
	getMarketDataFailureResponse           = "Market data retrieval failed due to internal error"
	getPairBalanceFailureResponse          = "Pair balance retrieval failed due to internal error"
	getPairBalanceNotFoundResponse         = "Balance not found"
	getOrdersFailureResponse               = "Orders retrieval failed due to internal error"

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
		c.JSON(http.StatusInternalServerError, gin.H{errorField: createOrderFailureResponse})
		deps.Logger.Error(err.Error())
		return
	}
	c.Status(http.StatusCreated)
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
