package http

import (
	"net/http"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/domain/usecase/deposit"
	"github.com/dixchange/Monolith/internal/domain/usecase/login"
	"github.com/dixchange/Monolith/internal/domain/usecase/register"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/pkg/errors"
)

const (
	invalidRequestParametersResponse = "Invalid request parameters"
	invalidCredentialsResponse       = "Invalid credentials" //nolint:gosec // False positive for hardcoded creds
	registrationFailureResponse      = "Registration failed due to internal error"
	depositFailureResponse           = "Deposit failed due to internal error"
	loginFailureResponse             = "Login failed due to internal error"

	errorField = "error"

	userIDParamKey = "user_id"
	loginOKField   = "ok"
)

func loginHandler(c *gin.Context, deps *Dependencies) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.Logger.Errorf(err.Error())
		return
	}

	output, err := deps.Login.Execute(c, login.Input{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.Logger.Errorf(err.Error())
			return
		}

		if errors.Is(err, login.ErrInvalidCredentials) {
			c.JSON(http.StatusForbidden, gin.H{errorField: invalidCredentialsResponse})
			deps.Logger.Errorf(err.Error())
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{errorField: loginFailureResponse})
		deps.Logger.Errorf(err.Error())
		return
	}
	setAuthCookies(c, output.AccessToken)
	c.JSON(http.StatusOK, gin.H{loginOKField: true})
}

func registerHandler(c *gin.Context, deps *Dependencies) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.Logger.Errorf(err.Error())
		return
	}
	_, err := deps.Register.Execute(c, register.Input{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.Logger.Errorf(err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{errorField: registrationFailureResponse})
		deps.Logger.Errorf(err.Error())
		return
	}

	c.Status(http.StatusCreated)
}

func depositHandler(c *gin.Context, deps *Dependencies) {
	var req DepositRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.Logger.Errorf(err.Error())
		return
	}

	userUID, err := uuid.Parse(c.Params.ByName(userIDParamKey))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
		deps.Logger.Errorf("invalid param user_id: %w", err.Error())
		return
	}

	_, err = deps.Deposit.Execute(c, deposit.Input{
		UserID: userUID,
		Amount: req.Amount,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.Logger.Errorf(err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{errorField: depositFailureResponse})
		deps.Logger.Errorf(err.Error())
		return
	}
	c.Status(http.StatusOK)
}
