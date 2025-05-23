package services

import (
	"log"
	"math/rand"
	"time"
)

// MockImageService provides a mock implementation for image generation
// This is useful for testing when the Replicate API is not available
type MockImageService struct {
	// Placeholder URLs for testing
	mockImageURLs []string
	random        *rand.Rand
}

// NewMockImageService creates a new mock image service
func NewMockImageService() *MockImageService {
	// Create a properly seeded random number generator
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)
	
	return &MockImageService{
		mockImageURLs: []string{
			"https://images.unsplash.com/photo-1698778573682-346d219402b5?q=80&w=512&h=512&auto=format&fit=crop",
			"https://images.unsplash.com/photo-1682687982167-d7fb3ed8541d?q=80&w=512&h=512&auto=format&fit=crop",
			"https://images.unsplash.com/photo-1682687982093-4ca1a2bd9ae8?q=80&w=512&h=512&auto=format&fit=crop",
			"https://images.unsplash.com/photo-1682687982183-c2937a74257c?q=80&w=512&h=512&auto=format&fit=crop",
		},
		random: rng,
	}
}

// GenerateImage returns a mock image URL
func (s *MockImageService) GenerateImage(prompt string) (string, error) {
	log.Printf("Mock generating image with prompt: %s", prompt)
	
	// Simulate processing time
	time.Sleep(2 * time.Second)
	
	// Return a random mock image URL
	randomIndex := s.random.Intn(len(s.mockImageURLs))
	
	log.Printf("Mock image generated: %s", s.mockImageURLs[randomIndex])
	return s.mockImageURLs[randomIndex], nil
}

// InpaintImage returns a mock inpainted image URL
func (s *MockImageService) InpaintImage(imageURL, prompt, maskURL string) (string, error) {
	log.Printf("Mock inpainting image with prompt: %s", prompt)
	
	// Simulate processing time
	time.Sleep(2 * time.Second)
	
	// Return a random mock image URL
	randomIndex := s.random.Intn(len(s.mockImageURLs))
	
	log.Printf("Mock inpainted image: %s", s.mockImageURLs[randomIndex])
	return s.mockImageURLs[randomIndex], nil
} 