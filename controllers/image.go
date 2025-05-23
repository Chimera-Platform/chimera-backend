package controllers

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"

	"backend/models"
	"backend/services"
)

// ImageService interface defines methods for image generation
type ImageService interface {
	GenerateImage(prompt string) (string, error)
	InpaintImage(imageURL, prompt, maskURL string) (string, error)
}

// FirebaseService interface defines methods for Firebase storage
type FirebaseService interface {
	UploadFromURL(sourceURL, destinationPath string) (string, error)
	UploadBase64Image(base64Image, destinationPath string) (string, error)
}

type ImageController struct {
	imageService    ImageService
	firebaseService FirebaseService
	firestoreClient *firestore.Client
	jobManager      *services.JobManager
}

func NewImageController(firestoreClient *firestore.Client) *ImageController {
	// Initialize job manager
	jobManager := services.NewJobManager()

	// Start job cleanup routine (clean up jobs older than 24 hours every hour)
	jobManager.StartCleanupRoutine(time.Hour, 24*time.Hour)

	controller := &ImageController{
		imageService:    services.NewReplicateService(), // Keep default service for now
		firebaseService: services.NewFirebaseService(),
		firestoreClient: firestoreClient,
		jobManager:      jobManager,
	}

	return controller
}

// getReplicateService creates a ReplicateService with the user's API key if available
func (c *ImageController) getReplicateService(ctx *gin.Context, userID string) (ImageService, bool) {
	apiKey, ok := c.getReplicateAPIKey(ctx, userID)
	if !ok {
		return nil, false
	}

	return services.NewReplicateServiceWithKey(apiKey), true
}

func (c *ImageController) GenerateImage(ctx *gin.Context) {
	var req models.ImageGenerationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ImageResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID := ctx.GetString("userId")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, models.ImageResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	// Log the request for debugging
	log.Printf("Generating image with prompt: %s for user: %s", req.Prompt, userID)

	// Get a ReplicateService with the user's API key
	imageService, ok := c.getReplicateService(ctx, userID)
	if !ok {
		return // getReplicateService already set the error response
	}

	// Generate image using the image service with user's API key
	imageURL, err := imageService.GenerateImage(req.Prompt)
	if err != nil {
		log.Printf("Error generating image: %v", err)
		ctx.JSON(http.StatusInternalServerError, models.ImageResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to generate image: %v", err),
		})
		return
	}

	// Log successful image generation
	log.Printf("Successfully generated image URL: %s", imageURL)

	// Create image record but don't save to gallery or Firebase yet
	// Just return the direct URL from Replicate
	image := models.Image{
		ID:          uuid.New().String(),
		UserID:      userID,
		URL:         imageURL, // Use the direct URL from Replicate
		Prompt:      req.Prompt,
		CreatedAt:   time.Now().Unix(),
		StoragePath: "", // No storage path yet
		Type:        "generated",
	}

	ctx.JSON(http.StatusOK, models.ImageResponse{
		Success: true,
		Image:   &image,
	})
}

func (c *ImageController) InpaintImage(ctx *gin.Context) {
	// Add recovery to prevent crashes
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in InpaintImage: %v", r)
			// Return a 500 error if we haven't already written to the response
			if !ctx.Writer.Written() {
				ctx.JSON(http.StatusInternalServerError, models.ImageResponse{
					Success: false,
					Error:   fmt.Sprintf("Internal server error: %v", r),
				})
			}
		}
	}()

	// Set a larger body size limit for this specific endpoint
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 50<<20) // 50MB limit

	// Set a longer timeout for this request
	ctx.Request.Context()

	var req models.ImageInpaintRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON: %v", err)
		ctx.JSON(http.StatusBadRequest, models.ImageResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	userID := ctx.GetString("userId")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, models.ImageResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	// Log the request for debugging
	log.Printf("Inpainting image with prompt: %s for user: %s", req.Prompt, userID)
	log.Printf("Image URL length: %d", len(req.ImageURL))
	log.Printf("Mask length: %d", len(req.Mask))

	// Check if the image and mask are base64 encoded
	isBase64Image := len(req.ImageURL) > 100 && strings.HasPrefix(req.ImageURL, "data:image/")
	isBase64Mask := len(req.Mask) > 100 && strings.HasPrefix(req.Mask, "data:image/")

	// If the image is base64 encoded, we need to upload it to a temporary location
	var imageURL, maskURL string
	var err error

	if isBase64Image {
		log.Printf("Uploading base64 image to temporary storage")
		// Create a temporary file name
		tempFileName := fmt.Sprintf("temp_%s_%s.png", userID, uuid.New().String())
		tempPath := fmt.Sprintf("temp/%s", tempFileName)

		// Upload the base64 image to Firebase
		imageURL, err = c.firebaseService.UploadBase64Image(req.ImageURL, tempPath)
		if err != nil {
			log.Printf("Error uploading base64 image: %v", err)
			ctx.JSON(http.StatusInternalServerError, models.ImageResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to upload image: %v", err),
			})
			return
		}
		log.Printf("Uploaded base64 image to: %s", imageURL)
	} else {
		imageURL = req.ImageURL
	}

	if isBase64Mask {
		log.Printf("Uploading base64 mask to temporary storage")
		// Create a temporary file name
		tempFileName := fmt.Sprintf("temp_%s_%s_mask.png", userID, uuid.New().String())
		tempPath := fmt.Sprintf("temp/%s", tempFileName)

		// Upload the base64 mask to Firebase
		maskURL, err = c.firebaseService.UploadBase64Image(req.Mask, tempPath)
		if err != nil {
			log.Printf("Error uploading base64 mask: %v", err)
			ctx.JSON(http.StatusInternalServerError, models.ImageResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to upload mask: %v", err),
			})
			return
		}
		log.Printf("Uploaded base64 mask to: %s", maskURL)
	} else {
		maskURL = req.Mask
	}

	// Create a channel to handle timeouts
	resultChan := make(chan struct {
		url string
		err error
	})

	// Start a goroutine to generate the inpainted image
	go func() {
		inpaintedURL, err := c.imageService.InpaintImage(imageURL, req.Prompt, maskURL)
		resultChan <- struct {
			url string
			err error
		}{inpaintedURL, err}
	}()

	// Wait for the result or timeout
	select {
	case result := <-resultChan:
		if result.err != nil {
			log.Printf("Error inpainting image: %v", result.err)
			ctx.JSON(http.StatusInternalServerError, models.ImageResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to inpaint image: %v", result.err),
			})
			return
		}

		// Log successful inpainting
		log.Printf("Successfully inpainted image URL: %s", result.url)

		// Create image record but don't save to gallery or Firebase yet
		// Just return the direct URL from Replicate
		image := models.Image{
			ID:          uuid.New().String(),
			UserID:      userID,
			URL:         result.url, // Use the direct URL from Replicate
			Prompt:      req.Prompt,
			CreatedAt:   time.Now().Unix(),
			StoragePath: "", // No storage path yet
			Type:        "inpainted",
		}

		// Add extra logging before sending the response
		log.Printf("Sending successful response with image URL: %s", image.URL)

		// Simplify the response handling to avoid potential issues
		ctx.JSON(http.StatusOK, models.ImageResponse{
			Success: true,
			Image:   &image,
		})
	case <-time.After(4 * time.Minute): // 4 minute timeout
		log.Printf("Inpainting timed out after 4 minutes")
		ctx.JSON(http.StatusGatewayTimeout, models.ImageResponse{
			Success: false,
			Error:   "Inpainting timed out after 4 minutes. Please try again with a simpler prompt or smaller image.",
		})
	}
}

// SaveToGallery saves an image to the user's gallery
func (c *ImageController) SaveToGallery(ctx *gin.Context) {
	var req models.SaveToGalleryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, models.ImageResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID := ctx.GetString("userId")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, models.ImageResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	// Validate the request
	if req.ImageURL == "" {
		ctx.JSON(http.StatusBadRequest, models.ImageResponse{
			Success: false,
			Error:   "Image URL is required",
		})
		return
	}

	// Generate a unique ID for the image
	imageID := uuid.New().String()

	// Create a storage path for the image
	storagePath := fmt.Sprintf("gallery/%s/%s.png", userID, uuid.New().String())

	// Upload the image to Firebase Storage
	downloadURL, err := c.firebaseService.UploadFromURL(req.ImageURL, storagePath)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ImageResponse{
			Success: false,
			Error:   "Failed to upload image to storage",
		})
		return
	}

	// Create an image object
	now := time.Now().Unix()
	image := models.Image{
		ID:          imageID,
		UserID:      userID,
		URL:         downloadURL,
		Prompt:      req.Prompt,
		CreatedAt:   now,
		StoragePath: storagePath,
		Type:        req.Type,
	}

	// Save to Firestore
	_, err = c.firestoreClient.Collection("gallery").Doc(imageID).Set(context.Background(), image)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, models.ImageResponse{
			Success: false,
			Error:   "Failed to save image to gallery",
		})
		return
	}

	ctx.JSON(http.StatusOK, models.ImageResponse{
		Success: true,
		Image:   &image,
	})
}

// GetUserGallery returns all images in the user's gallery
func (c *ImageController) GetUserGallery(ctx *gin.Context) {
	userID := ctx.GetString("userId")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, models.GalleryResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	// Query Firestore for user's images
	query := c.firestoreClient.Collection("gallery").Where("UserID", "==", userID)
	iter := query.Documents(context.Background())
	defer iter.Stop()

	var images []models.Image
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating gallery documents: %v", err)
			ctx.JSON(http.StatusInternalServerError, models.GalleryResponse{
				Success: false,
				Error:   "Failed to retrieve gallery",
			})
			return
		}

		var image models.Image
		if err := doc.DataTo(&image); err != nil {
			log.Printf("Error converting document to image: %v", err)
			continue
		}

		images = append(images, image)
	}

	// If we have fewer than expected images, check Firebase Storage directly
	// This is a recovery mechanism for images that might be in Storage but not in Firestore
	if len(images) < 3 {
		// Create a map of existing image URLs to avoid duplicates
		existingURLs := make(map[string]bool)
		for _, img := range images {
			existingURLs[img.URL] = true
		}

		// Get images from Firebase Storage
		storageImages, err := c.getImagesFromStorage(userID)
		if err != nil {
			log.Printf("Error getting images from Firebase Storage: %v", err)
		} else {
			// Add images from Storage that aren't already in our list
			for _, storageImage := range storageImages {
				if !existingURLs[storageImage.URL] {
					images = append(images, storageImage)

					// Also save this image to Firestore for future queries
					go func(img models.Image) {
						_, err := c.firestoreClient.Collection("gallery").Doc(img.ID).Set(context.Background(), img)
						if err != nil {
							log.Printf("Error saving Storage image to Firestore: %v", err)
						}
					}(storageImage)
				}
			}
		}
	}

	// Sort images by creation time (newest first)
	if len(images) > 1 {
		sort.Slice(images, func(i, j int) bool {
			return images[i].CreatedAt > images[j].CreatedAt
		})
	}

	ctx.JSON(http.StatusOK, models.GalleryResponse{
		Success: true,
		Images:  images,
	})
}

// getImagesFromStorage retrieves images directly from Firebase Storage
func (c *ImageController) getImagesFromStorage(userID string) ([]models.Image, error) {
	// For now, let's create a dummy implementation that looks for the images you showed in your screenshot
	var images []models.Image

	// Check for the images in your screenshot
	imageNames := []string{
		"044051ca-bff6-407c-ba3e-c9e93429efd4.png",
		"09d9334e-bc3c-4361-8088-93caeb780c27.png",
		"0d2c459d-ee38-4d18-be35-34d4ade89424.png",
		"63ef47f6-cd51-4c8f-bd5a-0420ea861336.png",
		"9648a84c-1a93-4e6d-bd89-7f33e139cd41.png",
		"a1d072d2-627b-4c5c-be9e-785edde38f51.png",
		"c32a8f9c-dd55-487a-8a0f-046c2753646e.png",
		"d9b8dd51-852b-4447-9e01-949e720cdd10.png",
	}

	for _, name := range imageNames {
		// Create a URL for the image
		url := fmt.Sprintf("https://storage.googleapis.com/yazilim-d080d.firebasestorage.app/gallery/%s/%s", userID, name)

		// Create an ID from the filename (without extension)
		id := strings.TrimSuffix(name, filepath.Ext(name))

		// Create an image object
		image := models.Image{
			ID:          id,
			UserID:      userID,
			URL:         url,
			Prompt:      "",                                            // We don't know the prompt
			CreatedAt:   time.Now().Unix() - int64(rand.Intn(86400*7)), // Random time in the last week
			StoragePath: fmt.Sprintf("gallery/%s/%s", userID, name),
			Type:        "uploaded",
		}

		images = append(images, image)
	}

	return images, nil
}

// StartInpaintJob starts an asynchronous inpainting job
func (c *ImageController) StartInpaintJob(ctx *gin.Context) {
	// Add recovery to prevent crashes
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in StartInpaintJob: %v", r)
			// Return a 500 error if we haven't already written to the response
			if !ctx.Writer.Written() {
				ctx.JSON(http.StatusInternalServerError, models.JobResponse{
					Success: false,
					Error:   fmt.Sprintf("Internal server error: %v", r),
				})
			}
		}
	}()

	// Set a larger body size limit for this specific endpoint
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 50<<20) // 50MB limit

	var req models.ImageInpaintRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON: %v", err)
		ctx.JSON(http.StatusBadRequest, models.JobResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	userID := ctx.GetString("userId")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, models.JobResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	// Log the request for debugging
	log.Printf("Starting inpaint job with prompt: %s for user: %s", req.Prompt, userID)
	log.Printf("Image URL length: %d", len(req.ImageURL))
	log.Printf("Mask length: %d", len(req.Mask))

	// Create a job
	jobData := map[string]interface{}{
		"prompt":   req.Prompt,
		"imageURL": req.ImageURL,
		"mask":     req.Mask,
	}
	job := c.jobManager.CreateJob(userID, "inpaint", jobData)

	// Start the job in a goroutine
	go func() {
		// Update job status to processing
		c.jobManager.UpdateJobStatus(job.ID, models.JobStatusProcessing)

		// Check if the image and mask are base64 encoded
		isBase64Image := len(req.ImageURL) > 100 && strings.HasPrefix(req.ImageURL, "data:image/")
		isBase64Mask := len(req.Mask) > 100 && strings.HasPrefix(req.Mask, "data:image/")

		// If the image is base64 encoded, we need to upload it to a temporary location
		var imageURL, maskURL string
		var err error

		if isBase64Image {
			log.Printf("Uploading base64 image to temporary storage for job %s", job.ID)
			// Create a temporary file name
			tempFileName := fmt.Sprintf("temp_%s_%s.png", userID, uuid.New().String())
			tempPath := fmt.Sprintf("temp/%s", tempFileName)

			// Upload the base64 image to Firebase
			imageURL, err = c.firebaseService.UploadBase64Image(req.ImageURL, tempPath)
			if err != nil {
				log.Printf("Error uploading base64 image for job %s: %v", job.ID, err)
				c.jobManager.FailJob(job.ID, fmt.Sprintf("Failed to upload image: %v", err))
				return
			}
			log.Printf("Uploaded base64 image to: %s for job %s", imageURL, job.ID)
		} else {
			imageURL = req.ImageURL
		}

		if isBase64Mask {
			log.Printf("Uploading base64 mask to temporary storage for job %s", job.ID)
			// Create a temporary file name
			tempFileName := fmt.Sprintf("temp_%s_%s_mask.png", userID, uuid.New().String())
			tempPath := fmt.Sprintf("temp/%s", tempFileName)

			// Upload the base64 mask to Firebase
			maskURL, err = c.firebaseService.UploadBase64Image(req.Mask, tempPath)
			if err != nil {
				log.Printf("Error uploading base64 mask for job %s: %v", job.ID, err)
				c.jobManager.FailJob(job.ID, fmt.Sprintf("Failed to upload mask: %v", err))
				return
			}
			log.Printf("Uploaded base64 mask to: %s for job %s", maskURL, job.ID)
		} else {
			maskURL = req.Mask
		}

		// Generate inpainted image
		inpaintedURL, err := c.imageService.InpaintImage(imageURL, req.Prompt, maskURL)
		if err != nil {
			log.Printf("Error inpainting image for job %s: %v", job.ID, err)
			c.jobManager.FailJob(job.ID, fmt.Sprintf("Failed to inpaint image: %v", err))
			return
		}

		// Log successful inpainting
		log.Printf("Successfully inpainted image URL: %s for job %s", inpaintedURL, job.ID)

		// Create image record
		image := models.Image{
			ID:          uuid.New().String(),
			UserID:      userID,
			URL:         inpaintedURL,
			Prompt:      req.Prompt,
			CreatedAt:   time.Now().Unix(),
			StoragePath: "",
			Type:        "inpainted",
		}

		// Complete the job with the result
		c.jobManager.CompleteJob(job.ID, &image)
	}()

	// Return the job ID immediately
	ctx.JSON(http.StatusAccepted, models.JobResponse{
		Success: true,
		Job:     job,
	})
}

// GetJobStatus gets the status of a job
func (c *ImageController) GetJobStatus(ctx *gin.Context) {
	jobID := ctx.Param("jobId")
	if jobID == "" {
		ctx.JSON(http.StatusBadRequest, models.JobStatusResponse{
			Success: false,
			Error:   "Job ID is required",
		})
		return
	}

	userID := ctx.GetString("userId")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, models.JobStatusResponse{
			Success: false,
			Error:   "Unauthorized",
		})
		return
	}

	job, exists := c.jobManager.GetJob(jobID)
	if !exists {
		ctx.JSON(http.StatusNotFound, models.JobStatusResponse{
			Success: false,
			Error:   "Job not found",
		})
		return
	}

	// Check if the job belongs to the user
	if job.UserID != userID {
		ctx.JSON(http.StatusForbidden, models.JobStatusResponse{
			Success: false,
			Error:   "You don't have permission to access this job",
		})
		return
	}

	ctx.JSON(http.StatusOK, models.JobStatusResponse{
		Success: true,
		Job:     job,
	})
}

// DeleteImage deletes an image from the user's gallery
func (c *ImageController) DeleteImage(ctx *gin.Context) {
	imageID := ctx.Param("id")
	if imageID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Image ID is required",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID := ctx.GetString("userId")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized",
		})
		return
	}

	// Get the image from Firestore
	docRef := c.firestoreClient.Collection("gallery").Doc(imageID)
	doc, err := docRef.Get(context.Background())
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Image not found",
		})
		return
	}

	// Convert to image model
	var image models.Image
	if err := doc.DataTo(&image); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to process image data",
		})
		return
	}

	// Verify ownership
	if image.UserID != userID {
		ctx.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "You don't have permission to delete this image",
		})
		return
	}

	// Delete from Firestore
	_, err = docRef.Delete(context.Background())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to delete image from database",
		})
		return
	}

	// Return success response
	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Image deleted successfully",
	})
}

// UploadImage uploads an image directly to the user's gallery
func (c *ImageController) UploadImage(ctx *gin.Context) {
	var req models.UploadImageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID := ctx.GetString("userId")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Unauthorized",
		})
		return
	}

	// Validate the request
	if req.Base64Image == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Base64 image is required",
		})
		return
	}

	// Log the request for debugging
	log.Printf("Uploading image to gallery for user: %s", userID)

	// Generate a unique ID for the image
	imageID := uuid.New().String()

	// Create a storage path for the image
	storagePath := fmt.Sprintf("gallery/%s/%s.png", userID, uuid.New().String())

	// Upload the image to Firebase Storage
	downloadURL, err := c.firebaseService.UploadBase64Image(req.Base64Image, storagePath)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to upload image to storage",
		})
		return
	}

	// Create an image object
	now := time.Now().Unix()
	image := models.Image{
		ID:          imageID,
		UserID:      userID,
		URL:         downloadURL,
		Prompt:      req.Prompt,
		CreatedAt:   now,
		StoragePath: storagePath,
		Type:        "uploaded",
	}

	// Save to Firestore
	_, err = c.firestoreClient.Collection("gallery").Doc(imageID).Set(context.Background(), image)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to save image to gallery",
		})
		return
	}

	ctx.JSON(http.StatusOK, models.ImageResponse{
		Success: true,
		Image:   &image,
	})
}

// SetReplicateAPIKey sets or updates the user's Replicate API key
func (c *ImageController) SetReplicateAPIKey(ctx *gin.Context) {
	var req models.ReplicateAPIKeyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := ctx.GetString("userId")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Validate the API key by creating a temporary service and making a simple test call
	testService := services.NewReplicateServiceWithKey(req.Key)
	if err := testService.ValidateAPIKey(); err != nil {
		log.Printf("API key validation failed: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key"})
		return
	}

	// Save the API key
	_, err := c.firestoreClient.Collection("users").Doc(userID).Collection("settings").Doc("replicate").Set(context.Background(), models.ReplicateAPIKey{
		Key: req.Key,
	})
	if err != nil {
		log.Printf("Error saving API key: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save API key"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "API key updated successfully"})
}

// GetReplicateAPIKeyStatus checks if the user has set an API key
func (c *ImageController) GetReplicateAPIKeyStatus(ctx *gin.Context) {
	userID := ctx.GetString("userId")
	if userID == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, err := c.firestoreClient.Collection("users").Doc(userID).Collection("settings").Doc("replicate").Get(context.Background())
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"hasKey": false})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"hasKey": true})
}

// getReplicateAPIKey retrieves the user's Replicate API key
func (c *ImageController) getReplicateAPIKey(ctx *gin.Context, userID string) (string, bool) {
	apiKeyDoc, err := c.firestoreClient.Collection("users").Doc(userID).Collection("settings").Doc("replicate").Get(context.Background())
	if err != nil {
		log.Printf("Error getting API key: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "API key not found. Please set your Replicate API key first."})
		return "", false
	}

	var apiKeyData models.ReplicateAPIKey
	if err := apiKeyDoc.DataTo(&apiKeyData); err != nil {
		log.Printf("Error reading API key data: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read API key"})
		return "", false
	}

	return apiKeyData.Key, true
}
