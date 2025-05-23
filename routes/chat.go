package routes

import (
	"backend/controllers"
	"backend/middleware"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
)

func SetupChatRoutes(router *gin.Engine, firestoreClient *firestore.Client) {
	chatController := controllers.NewChatController(firestoreClient)

	// Chat routes (all protected by auth)
	chatGroup := router.Group("/api/chat")
	chatGroup.Use(middleware.AuthMiddleware())
	{
		// Chat management
		chatGroup.POST("/create", chatController.CreateChat)
		chatGroup.POST("/:chatID/message", chatController.SendMessage)
		chatGroup.POST("/:chatID/message/stream", chatController.SendMessageStream)
		chatGroup.GET("/list", chatController.GetChats)
		chatGroup.GET("/:chatID", chatController.GetChat)
		chatGroup.DELETE("/:chatID", chatController.DeleteChat)

		// OpenRouter configuration
		chatGroup.GET("/models", chatController.GetModels)
		chatGroup.POST("/apikey", chatController.SetAPIKey)
		chatGroup.GET("/apikey/status", chatController.GetAPIKeyStatus)
		chatGroup.GET("/credits", chatController.GetCredits) // Add new endpoint for credits
	}
}
