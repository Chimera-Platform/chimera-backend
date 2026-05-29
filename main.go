package main

import (
	"log"
	"os"

	"backend/firebase"
	"backend/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// On hosts without a service account file (e.g. Railway), materialize the
	// credentials from FIREBASE_SERVICE_ACCOUNT_JSON into a file and point the
	// credential env vars at it. No-op locally when the file already exists.
	if err := writeServiceAccountFromEnv(); err != nil {
		log.Fatalf("Failed to write service account from env: %v", err)
	}

	// Initialize Firebase
	if err := firebase.InitFirebase(); err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}

	// Ensure Firebase Auth client is initialized
	authClient := firebase.GetAuthClient()
	if authClient == nil {
		log.Fatalf("Firebase Auth client is nil")
	}

	// Set up Gin router
	router := gin.Default()

	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:8080", "*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Add Firebase Auth client to context
	router.Use(func(c *gin.Context) {
		c.Set("firebaseAuth", authClient)
		c.Next()
	})

	// Set up routes
	routes.SetupRouter(router)

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// writeServiceAccountFromEnv writes the service account JSON provided via the
// FIREBASE_SERVICE_ACCOUNT_JSON env var to a local file and points the
// credential env vars (used by Auth/Firestore and Storage) at it. If the env
// var is empty it does nothing, so local setups using a real file keep working.
func writeServiceAccountFromEnv() error {
	jsonContent := os.Getenv("FIREBASE_SERVICE_ACCOUNT_JSON")
	if jsonContent == "" {
		return nil
	}

	const path = "serviceAccountKey.json"
	if err := os.WriteFile(path, []byte(jsonContent), 0600); err != nil {
		return err
	}

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", path)
	os.Setenv("FIREBASE_SERVICE_ACCOUNT_KEY", path)
	log.Printf("Wrote service account credentials from FIREBASE_SERVICE_ACCOUNT_JSON to %s", path)
	return nil
}
