package services

import (
	"fmt"
	"log"
	"time"
)

// MockFirebaseService provides a mock implementation for Firebase storage
// This is useful for testing when Firebase is not properly configured
type MockFirebaseService struct {
	// Base URL for mock images
	baseURL string
}

// NewMockFirebaseService creates a new mock Firebase service
func NewMockFirebaseService() *MockFirebaseService {
	return &MockFirebaseService{
		baseURL: "https://via.placeholder.com",
	}
}

// UploadFromURL simulates uploading an image to Firebase Storage
// Instead of actually uploading, it returns a placeholder URL
func (s *MockFirebaseService) UploadFromURL(sourceURL, destinationPath string) (string, error) {
	log.Printf("Mock uploading image from %s to path: %s", sourceURL, destinationPath)
	
	// Simulate processing time
	time.Sleep(1 * time.Second)
	
	// Generate a mock Firebase Storage URL
	// Format: https://via.placeholder.com/512x512?text=Mock+Image+{timestamp}
	timestamp := time.Now().Unix()
	mockURL := fmt.Sprintf("%s/512x512?text=Mock+Image+%d", s.baseURL, timestamp)
	
	log.Printf("Mock upload complete. URL: %s", mockURL)
	return mockURL, nil
} 