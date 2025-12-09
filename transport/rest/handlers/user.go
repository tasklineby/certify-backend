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

// CreateCompanyWithAdmin godoc
// @Summary      Create company with admin
// @Description  Create a new company and register its admin user. Returns access and refresh tokens for the admin.
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        request   body      entity.CreateCompanyRequest  true  "Company and admin data"
// @Success      201       {object}  entity.TokenPair             "Company created, admin registered"
// @Failure      400       {object}  errs.Error                   "Invalid request"
// @Failure      409       {object}  errs.Error                   "Email already exists"
// @Router       /user/company [post]
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

// GetUser godoc
// @Summary      Get user by ID
// @Description  Get user profile information by user ID
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id        path      int            true  "User ID"
// @Success      200       {object}  entity.User    "User profile"
// @Failure      400       {object}  errs.Error     "Invalid user ID"
// @Failure      401       {object}  errs.Error     "Unauthorized"
// @Failure      404       {object}  errs.Error     "User not found"
// @Router       /user/{id} [get]
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

// GetMe godoc
// @Summary      Get current user
// @Description  Get the authenticated user's profile information
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200       {object}  entity.User      "User profile"
// @Failure      401       {object}  errs.Error       "Unauthorized"
// @Failure      404       {object}  errs.Error       "User not found"
// @Router       /user/me [get]
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

// UpdateUser godoc
// @Summary      Update user by ID
// @Description  Update user profile information. Only admins can update other users from the same company.
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id        path      int                    true  "User ID"
// @Param        request   body      entity.UpdateUserRequest  true  "User update data"
// @Success      200       {object}  map[string]string      "User updated successfully"
// @Failure      400       {object}  errs.Error             "Invalid request"
// @Failure      401       {object}  errs.Error             "Unauthorized - only admins can update other users"
// @Failure      404       {object}  errs.Error             "User not found"
// @Failure      409       {object}  errs.Error             "Email already exists"
// @Router       /user/{id} [put]
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

// UpdateMe godoc
// @Summary      Update current user
// @Description  Update the authenticated user's profile information
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request   body      entity.UpdateUserRequest  true  "User update data"
// @Success      200       {object}  map[string]string         "User updated successfully"
// @Failure      400       {object}  errs.Error               "Invalid request"
// @Failure      401       {object}  errs.Error               "Unauthorized"
// @Failure      404       {object}  errs.Error               "User not found"
// @Failure      409       {object}  errs.Error               "Email already exists"
// @Router       /user/me [put]
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

// DeleteUser godoc
// @Summary      Delete user by ID
// @Description  Delete a user. Only admins can delete users from the same company.
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id        path      int                true  "User ID"
// @Success      200       {object}  map[string]string  "User deleted successfully"
// @Failure      400       {object}  errs.Error         "Invalid user ID"
// @Failure      401       {object}  errs.Error         "Unauthorized - only admins can delete users"
// @Failure      404       {object}  errs.Error         "User not found"
// @Router       /user/{id} [delete]
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

// GetUsersByCompany godoc
// @Summary      Get users by company
// @Description  Get all users from the authenticated user's company
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200       {array}   entity.User    "List of users"
// @Failure      401       {object}  errs.Error     "Unauthorized"
// @Failure      404       {object}  errs.Error     "Company not found"
// @Router       /user/company [get]
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
