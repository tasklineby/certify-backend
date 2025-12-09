package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tasklineby/certify-backend/entity"
	"github.com/tasklineby/certify-backend/errs"
	"github.com/tasklineby/certify-backend/repository/rdb"
	"github.com/tasklineby/certify-backend/service"
)

type UserHandler struct {
	userService service.UserService
	jwtService  service.JwtService
	tokenRepo   rdb.TokenRepository
}

func NewUserHandler(userService service.UserService, jwtService service.JwtService, tokenRepo rdb.TokenRepository) *UserHandler {
	return &UserHandler{
		userService: userService,
		jwtService:  jwtService,
		tokenRepo:   tokenRepo,
	}
}

func (h *UserHandler) CreateCompanyWithAdmin(c *gin.Context) {
	var req entity.CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errs.BadRequestError("invalid request body", err))
		return
	}

	tokenPair, err := h.userService.CreateCompanyWithAdmin(c.Request.Context(), req, h.jwtService, h.tokenRepo)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusCreated, tokenPair)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errs.BadRequestError("invalid user ID", err))
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errs.UnauthorizedError("user ID not found", nil))
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID.(int))
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errs.BadRequestError("invalid user ID", err))
		return
	}

	requesterRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, errs.UnauthorizedError("user role not found", nil))
		return
	}

	companyIDStr, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errs.UnauthorizedError("company ID not found", nil))
		return
	}

	requesterCompanyID, err := strconv.Atoi(companyIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errs.InternalError("invalid company ID in token", err))
		return
	}

	var req entity.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errs.BadRequestError("invalid request body", err))
		return
	}

	requesterID, _ := c.Get("user_id")
	err = h.userService.UpdateUser(c.Request.Context(), userID, req, requesterRole.(string), requesterCompanyID, requesterID.(int))
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errs.UnauthorizedError("user ID not found", nil))
		return
	}

	// Get requester info from context (set by middleware)
	requesterRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, errs.UnauthorizedError("user role not found", nil))
		return
	}

	companyIDStr, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errs.UnauthorizedError("company ID not found", nil))
		return
	}

	requesterCompanyID, err := strconv.Atoi(companyIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errs.InternalError("invalid company ID in token", err))
		return
	}

	var req entity.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errs.BadRequestError("invalid request body", err))
		return
	}

	err = h.userService.UpdateUser(c.Request.Context(), userID.(int), req, requesterRole.(string), requesterCompanyID, userID.(int))
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get requester info from context (set by middleware)
	requesterRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
		return
	}

	companyIDStr, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Company ID not found in context"})
		return
	}

	requesterCompanyID, err := strconv.Atoi(companyIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid company ID in token"})
		return
	}

	err = h.userService.DeleteUser(c.Request.Context(), userID, requesterRole.(string), requesterCompanyID)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func (h *UserHandler) GetUsersByCompany(c *gin.Context) {
	// Get company_id from token payload (set by middleware)
	companyIDStr, exists := c.Get("company_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Company ID not found in context"})
		return
	}

	companyID, err := strconv.Atoi(companyIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, errs.InternalError("invalid company ID in token", err))
		return
	}

	users, err := h.userService.GetUsersByCompanyID(c.Request.Context(), companyID)
	if err != nil {
		errCast := errs.ErrorCast(err)
		c.JSON(errCast.StatusCode(), errCast)
		return
	}

	// Remove passwords from response
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, users)
}
