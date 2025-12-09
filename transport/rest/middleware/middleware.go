package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tasklineby/certify-backend/errs"
	"github.com/tasklineby/certify-backend/service"
)

func AuthMiddleware(authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, errs.UnauthorizedError("authorization header required", nil))
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" || token == authHeader {
			c.JSON(http.StatusUnauthorized, errs.UnauthorizedError("invalid token", nil))
			c.Abort()
			return
		}

		tokenPayload, err := authService.ParseToken(c.Request.Context(), token)
		if err != nil {
			errCast := errs.ErrorCast(err)
			c.JSON(errCast.StatusCode(), errCast)
			c.Abort()
			return
		}

		userID, _ := strconv.Atoi(tokenPayload.UserID)
		c.Set("user_id", userID)
		c.Set("user_role", tokenPayload.Role)
		c.Set("company_id", tokenPayload.CompanyID)
		c.Next()
	}
}
