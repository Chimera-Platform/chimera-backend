// This is a utility script to check if the configuration is correct
// Run it with: go run cmd/check_config/main.go

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	firebase "firebase.google.com/go/v4"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	} else {
		log.Println("Successfully loaded .env file")
	}

	fmt.Println("=== Configuration Check ===")
	fmt.Println("This utility checks if your configuration is correct.")
	fmt.Println("Choose what to check:")
	fmt.Println("1. Firebase Storage")
	fmt.Println("2. Replicate API")
	fmt.Println("3. Both")
	fmt.Println("4. Exit")

	var choice string
	fmt.Print("Enter your choice (1-4): ")
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		checkFirebase()
	case "2":
		checkReplicate()
	case "3":
		checkFirebase()
		fmt.Println("\n-----------------------------------\n")
		checkReplicate()
	case "4":
		fmt.Println("Exiting...")
		return
	default:
		fmt.Println("Invalid choice. Exiting...")
		return
	}
}

func checkFirebase() {
	fmt.Println("\n=== Firebase Configuration Check ===")
	
	// Get Firebase configuration from environment variables
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	bucketName := os.Getenv("FIREBASE_STORAGE_BUCKET")

	fmt.Printf("Project ID: %s\n", projectID)
	fmt.Printf("Storage Bucket: %s\n", bucketName)

	// Check service account key file
	serviceAccountPath := "serviceAccountKey.json"
	if _, err := os.Stat(serviceAccountPath); os.IsNotExist(err) {
		fmt.Printf("❌ Service account key file not found at %s\n", serviceAccountPath)
		fmt.Println("Please download your service account key from Firebase console:")
		fmt.Println("1. Go to Firebase console -> Project settings -> Service accounts")
		fmt.Println("2. Click 'Generate new private key'")
		fmt.Println("3. Save the file as 'serviceAccountKey.json' in the backend directory")
		return
	} else {
		fmt.Printf("✅ Service account key file found at %s\n", serviceAccountPath)
	}

	// Try to initialize Firebase
	ctx := context.Background()
	config := &firebase.Config{
		ProjectID:     projectID,
		StorageBucket: bucketName,
	}
	
	opt := option.WithCredentialsFile(serviceAccountPath)
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		fmt.Printf("❌ Failed to initialize Firebase app: %v\n", err)
		return
	}
	fmt.Println("✅ Firebase app initialized successfully")

	// Try to get Storage client
	client, err := app.Storage(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to create Storage client: %v\n", err)
		return
	}
	fmt.Println("✅ Storage client created successfully")

	// Try to get bucket
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		fmt.Printf("❌ Failed to get bucket: %v\n", err)
		return
	}
	fmt.Println("✅ Successfully connected to bucket")

	// Check if bucket exists
	_, err = bucket.Attrs(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to get bucket attributes: %v\n", err)
		fmt.Println("This usually means the bucket doesn't exist or you don't have permission to access it.")
		return
	}
	fmt.Println("✅ Bucket exists and is accessible")

	fmt.Println("\n✅ Firebase configuration is correct!")
}

func checkReplicate() {
	fmt.Println("\n=== Replicate API Configuration Check ===")
	
	// Get Replicate API key from environment variables
	apiKey := os.Getenv("REPLICATE_API_KEY")
	
	// Check if API key is set
	if apiKey == "" {
		fmt.Println("❌ REPLICATE_API_KEY environment variable is not set")
		fmt.Println("Please set your Replicate API key in the .env file")
		return
	}
	
	// Check if API key has the correct format
	if !strings.HasPrefix(apiKey, "r8_") {
		fmt.Println("⚠️ Warning: Replicate API key does not start with 'r8_', which is the expected format")
	} else {
		fmt.Println("✅ Replicate API key has the correct format")
	}
	
	fmt.Println("\n✅ Replicate API key is set correctly!")
	fmt.Println("To fully test the Replicate API, try generating an image from the application.")
} 