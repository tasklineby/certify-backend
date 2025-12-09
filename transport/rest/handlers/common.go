package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/tasklineby/certify-backend/service"
	"github.com/tasklineby/certify-backend/transport/rest/middleware"
)

func InitRoutes(
	userHandler *UserHandler,
	authHandler *AuthHandler,
	authService service.AuthService,
) *gin.Engine {
	router := gin.New()

	// Public routes
	api := router.Group("/api")

	// Auth routes (public)
	authApi := api.Group("/auth")
	authApi.POST("/login", authHandler.Login)
	authApi.POST("/register", authHandler.Register)
	authApi.POST("/refresh", authHandler.Refresh)
	authApi.POST("/logout", authHandler.Logout)

	// User routes (public for company creation)
	userApi := api.Group("/user")
	userApi.POST("/company", userHandler.CreateCompanyWithAdmin)

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(authService))

	// User routes (protected)
	protectedUserApi := protected.Group("/user")
	protectedUserApi.GET("/me", userHandler.GetMe)
	protectedUserApi.PUT("/me", userHandler.UpdateMe)
	protectedUserApi.GET("/:id", userHandler.GetUser)
	protectedUserApi.PUT("/:id", userHandler.UpdateUser)
	protectedUserApi.DELETE("/:id", userHandler.DeleteUser)
	protectedUserApi.GET("/company", userHandler.GetUsersByCompany)

	return router
}
