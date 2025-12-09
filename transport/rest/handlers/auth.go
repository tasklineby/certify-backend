package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tasklineby/certify-backend/entity"
	"github.com/tasklineby/certify-backend/errs"
	"github.com/tasklineby/certify-backend/service"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate user with email and password, returns access and refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request   body      entity.LoginRequest  true  "Login credentials"
// @Success      200       {object}  entity.TokenPair      "Successfully authenticated"
// @Failure      400       {object}  errs.Error            "Invalid request"
// @Failure      401       {object}  errs.Error            "Invalid credentials"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req entity.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenPair, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusOK, tokenPair)
}

// Register godoc
// @Summary      Register employee
// @Description  Register a new employee user and return access and refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request   body      entity.RegisterEmployeeRequest  true  "Employee registration data"
// @Success      201       {object}  entity.TokenPair                "Successfully registered"
// @Failure      400       {object}  errs.Error                     "Invalid request"
// @Failure      409       {object}  errs.Error                     "Email already exists"
// @Failure      404       {object}  errs.Error                     "Company not found"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req entity.RegisterEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenPair, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusCreated, tokenPair)
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Exchange refresh token for new access and refresh token pair
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request   body      entity.RefreshRequest  true  "Refresh token"
// @Success      200       {object}  entity.TokenPair        "New token pair"
// @Failure      400       {object}  errs.Error             "Invalid request"
// @Failure      401       {object}  errs.Error             "Invalid or expired refresh token"
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req entity.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenPair, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusOK, tokenPair)
}

// Logout godoc
// @Summary      Logout user
// @Description  Invalidate access and refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request   body      entity.RefreshRequest  true  "Refresh token"
// @Success      200       {object}  map[string]string      "Successfully logged out"
// @Failure      400       {object}  errs.Error             "Invalid request"
// @Failure      401       {object}  errs.Error             "Unauthorized"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization header required"})
		return
	}

	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	var req entity.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.authService.Logout(c.Request.Context(), accessToken, req.RefreshToken)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}
