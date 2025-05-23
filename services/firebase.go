package services

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/storage"
	"google.golang.org/api/option"
)

type FirebaseService struct {
	storageClient *storage.Client
	bucket        string
}

func NewFirebaseService() *FirebaseService {
	ctx := context.Background()

	// bulamazsa çöksün abi
	bucketName := os.Getenv("FIREBASE_STORAGE_BUCKET")
	if bucketName == "" {
		log.Fatalf("ERROR: FIREBASE_STORAGE_BUCKET environment variable must be set")
	}

	// Path to service account key file from environment
	serviceAccountPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if serviceAccountPath == "" {
		log.Printf("WARNING: GOOGLE_APPLICATION_CREDENTIALS environment variable is not set")
		// Return a service that will return errors when used
		return &FirebaseService{
			storageClient: nil,
			bucket:        bucketName,
		}
	}

	// Initialize Firebase app with service account
	config := &firebase.Config{
		StorageBucket: bucketName,
	}

	opt := option.WithCredentialsFile(serviceAccountPath)
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		log.Printf("ERROR: Failed to initialize Firebase app: %v", err)
		// Return a service that will return errors when used
		return &FirebaseService{
			storageClient: nil,
			bucket:        bucketName,
		}
	}

	// Get Storage client
	client, err := app.Storage(ctx)
	if err != nil {
		log.Printf("ERROR: Failed to create Storage client: %v", err)
		// Return a service that will return errors when used
		return &FirebaseService{
			storageClient: nil,
			bucket:        bucketName,
		}
	}

	log.Printf("Firebase Storage initialized successfully with bucket: %s", bucketName)
	return &FirebaseService{
		storageClient: client,
		bucket:        bucketName,
	}
}

func (s *FirebaseService) UploadFromURL(sourceURL, destinationPath string) (string, error) {
	// If storage client is nil, return error
	if s.storageClient == nil {
		return "", fmt.Errorf("Firebase Storage client not initialized")
	}

	ctx := context.Background()
	log.Printf("Uploading from URL %s to path %s", sourceURL, destinationPath)

	// Create a bucket handle
	bucket, err := s.storageClient.Bucket(s.bucket)
	if err != nil {
		return "", fmt.Errorf("failed to get bucket: %v", err)
	}

	// Create object handle
	obj := bucket.Object(destinationPath)

	// Download image from URL
	resp, err := http.Get(sourceURL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image, status code: %d", resp.StatusCode)
	}

	// Create writer
	writer := obj.NewWriter(ctx)
	writer.ContentType = "image/png" // Set content type

	// Copy the image data to Firebase Storage
	if _, err := io.Copy(writer, resp.Body); err != nil {
		return "", fmt.Errorf("failed to copy image to storage: %v", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %v", err)
	}

	// Make the object public
	if err := obj.ACL().Set(ctx, "allUsers", "READER"); err != nil {
		return "", fmt.Errorf("failed to make object public: %v", err)
	}

	// Get the public URL
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get object attributes: %v", err)
	}

	log.Printf("Successfully uploaded image to Firebase Storage: %s", attrs.MediaLink)
	return attrs.MediaLink, nil
}

// UploadBase64Image uploads a base64 encoded image to Firebase Storage
func (s *FirebaseService) UploadBase64Image(base64Image, destinationPath string) (string, error) {
	// If storage client is nil, return error
	if s.storageClient == nil {
		return "", fmt.Errorf("Firebase Storage client not initialized")
	}

	// Create a temporary directory if it doesn't exist
	tempDir := filepath.Join("temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	// Extract the base64 data
	var base64Data string
	var contentType string

	// Log the length of the incoming base64 string for debugging
	log.Printf("Received base64 image with length: %d", len(base64Image))

	if strings.HasPrefix(base64Image, "data:") {
		// Extract the base64 part from the data URL
		parts := strings.Split(base64Image, ",")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid base64 image format: wrong number of parts")
		}

		// Extract content type from the data URL
		metaParts := strings.Split(parts[0], ";")
		if len(metaParts) < 1 {
			return "", fmt.Errorf("invalid base64 image format: missing content type")
		}

		contentTypeParts := strings.Split(metaParts[0], ":")
		if len(contentTypeParts) != 2 {
			return "", fmt.Errorf("invalid base64 image format: invalid content type")
		}

		contentType = contentTypeParts[1]
		base64Data = parts[1]

		log.Printf("Extracted content type: %s", contentType)
	} else {
		// Assume it's already base64 data without the data URL prefix
		base64Data = base64Image
		contentType = "image/png" // Default to PNG
	}

	// Validate base64 data
	if len(base64Data) == 0 {
		return "", fmt.Errorf("empty base64 data")
	}

	// Check if the base64 data is valid
	if !isValidBase64(base64Data) {
		return "", fmt.Errorf("invalid base64 encoding")
	}

	// Decode the base64 data
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 image: %v", err)
	}

	ctx := context.Background()
	log.Printf("Uploading base64 image to path %s (size: %d bytes)", destinationPath, len(imageData))

	// Create a bucket handle
	bucket, err := s.storageClient.Bucket(s.bucket)
	if err != nil {
		return "", fmt.Errorf("failed to get bucket: %v", err)
	}

	// Create object handle
	obj := bucket.Object(destinationPath)

	// Create writer
	writer := obj.NewWriter(ctx)
	writer.ContentType = contentType // Set content type

	// Copy the image data to Firebase Storage
	if _, err := writer.Write(imageData); err != nil {
		return "", fmt.Errorf("failed to write image to storage: %v", err)
	}

	// Close the writer
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %v", err)
	}

	// Make the object public
	if err := obj.ACL().Set(ctx, "allUsers", "READER"); err != nil {
		return "", fmt.Errorf("failed to make object public: %v", err)
	}

	// Get the public URL
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get object attributes: %v", err)
	}

	log.Printf("Successfully uploaded base64 image to Firebase Storage: %s", attrs.MediaLink)
	return attrs.MediaLink, nil
}

// isValidBase64 checks if a string is valid base64 encoding
func isValidBase64(s string) bool {
	// Check if the string length is valid for base64
	if len(s)%4 != 0 {
		// Try to add padding if missing
		s = s + strings.Repeat("=", 4-len(s)%4)
	}

	// Check if the string contains only valid base64 characters
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
			return false
		}
	}

	// Try to decode a small part to validate
	_, err := base64.StdEncoding.DecodeString(s[:min(100, len(s))])
	return err == nil
}

// min returns the smaller of x or y
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
