package http

import (
	"net/http"

	"github.com/dixchange/Monolith/internal/auth/domain/entity"
	"github.com/dixchange/Monolith/internal/auth/domain/usecase/login"
	"github.com/dixchange/Monolith/internal/auth/domain/usecase/register"
	"github.com/gin-gonic/gin"

	"github.com/pkg/errors"
)

const (
	invalidRequestParametersResponse = "Invalid request parameters"
	invalidCredentialsResponse       = "Invalid credentials" //nolint:gosec // False positive for hardcoded creds
	registrationFailureResponse      = "Registration failed due to internal error"
	loginFailureResponse             = "Login failed due to internal error"

	errorField = "error"
)

func loginHandler(c *gin.Context, deps *AuthDependencies) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.logger.Errorf(err.Error())
		return
	}

	output, err := deps.log.Execute(c, login.Input{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.logger.Errorf(err.Error())
			return
		}

		if errors.Is(err, login.ErrInvalidCredentials) {
			c.JSON(http.StatusForbidden, gin.H{errorField: invalidCredentialsResponse})
			deps.logger.Errorf(err.Error())
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{errorField: loginFailureResponse})
		deps.logger.Errorf(err.Error())
		return
	}
	c.JSON(http.StatusOK, LoginResponse{AccessToken: output.AccessToken})
}

func registerHandler(c *gin.Context, deps *AuthDependencies) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		deps.logger.Errorf(err.Error())
		return
	}
	_, err := deps.reg.Execute(c, register.Input{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			deps.logger.Errorf(err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{errorField: registrationFailureResponse})
		deps.logger.Errorf(err.Error())
		return
	}

	c.Status(http.StatusCreated)
}
