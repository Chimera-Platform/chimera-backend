package routes

import (
	"backend/controllers"
	"backend/firebase"
	"backend/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter sets up all the API routes.
func SetupRouter(router *gin.Engine) {
	// Public endpoints.
	api := router.Group("/api")
	{
		api.GET("/ping", controllers.PingHandler)
	}

	// Protected endpoints (require authentication).
	protected := router.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		// Setup profile routes
		protected.GET("/profile", controllers.ProfileHandler)

		// Setup inpaint route
		protected.POST("/inpaint", controllers.InpaintHandler)

		// Setup avatar routes
		avatarController := controllers.NewAvatarController(firebase.GetFirestoreClient())

		// Public avatar endpoints (optional auth)
		publicAvatars := router.Group("/api/avatars")
		publicAvatars.Use(middleware.OptionalAuthMiddleware())
		{
			publicAvatars.GET("/public", avatarController.GetPublicAvatars)
		}

		// Protected avatar endpoints (require authentication)
		protectedAvatars := router.Group("/api/avatars")
		protectedAvatars.Use(middleware.AuthMiddleware())
		{
			protectedAvatars.GET("/user", avatarController.GetUserAvatars)
			protectedAvatars.GET("/:id", avatarController.GetAvatar)
			protectedAvatars.POST("", avatarController.CreateAvatar)
			protectedAvatars.PUT("/:id", avatarController.UpdateAvatar)
			protectedAvatars.DELETE("/:id", avatarController.DeleteAvatar)
		}
	}

	// Setup image routes (using the dedicated function)
	SetupImageRoutes(router)

	// Setup chat routes
	SetupChatRoutes(router, firebase.GetFirestoreClient())
}
