package http

import (
	"net/http"

	"github.com/dixchange/Monolith/internal/auth/domain/entity"
	"github.com/dixchange/Monolith/internal/auth/domain/usecase/register"
	"github.com/gin-gonic/gin"

	"github.com/pkg/errors"
)

const (
	invalidRequestParametersResponse = "Invalid request parameters"
	registrationFailureResponse      = "Registration failed due to internal error"

	errorField = "error"
)

func loginHandler(c *gin.Context, _ *AuthDependencies) {
	c.JSON(http.StatusOK, gin.H{"action": "login"})
}

func registerHandler(c *gin.Context, deps *AuthDependencies) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{errorField: err.Error()})
		return
	}
	_, err := deps.reg.Execute(c, register.Input{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.As(err, &entity.ValidationError{}) {
			c.JSON(http.StatusBadRequest, gin.H{errorField: invalidRequestParametersResponse})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{errorField: registrationFailureResponse})
		return
	}

	c.Status(http.StatusOK)
}
