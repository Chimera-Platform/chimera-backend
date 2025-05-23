package routes

import (
	"backend/controllers"
	"backend/firebase"
	"backend/middleware"

	"github.com/gin-gonic/gin"
)

func SetupImageRoutes(router *gin.Engine) {
	imageController := controllers.NewImageController(firebase.GetFirestoreClient())

	// Group image routes with auth middleware
	imageRoutes := router.Group("/api/images")
	imageRoutes.Use(middleware.AuthMiddleware())
	{
		// Image generation endpoints
		imageRoutes.POST("/generate", imageController.GenerateImage)
		imageRoutes.POST("/inpaint", imageController.InpaintImage)

		// Gallery endpoints
		imageRoutes.POST("/gallery", imageController.SaveToGallery)
		imageRoutes.GET("/gallery", imageController.GetUserGallery)
		imageRoutes.DELETE("/gallery/:id", imageController.DeleteImage)
		imageRoutes.POST("/upload", imageController.UploadImage)

		// Job-based endpoints
		imageRoutes.POST("/jobs/inpaint", imageController.StartInpaintJob)
		imageRoutes.GET("/jobs/:jobId", imageController.GetJobStatus)

		// API key management endpoints
		imageRoutes.POST("/apikey", imageController.SetReplicateAPIKey)
		imageRoutes.GET("/apikey/status", imageController.GetReplicateAPIKeyStatus)
	}
}
