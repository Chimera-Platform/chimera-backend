package models

type Image struct {
	ID          string `json:"id"`
	UserID      string `json:"userId"`
	URL         string `json:"url"`
	Prompt      string `json:"prompt"`
	CreatedAt   int64  `json:"createdAt"`
	StoragePath string `json:"storagePath"`
	Type        string `json:"type"` // "generated", "inpainted", "variation"
}

type ImageGenerationRequest struct {
	Prompt string `json:"prompt"`
}

type ImageInpaintRequest struct {
	ImageURL string `json:"imageUrl"`
	Prompt   string `json:"prompt"`
	Mask     string `json:"mask"`
}

type ImageVariationRequest struct {
	ImageURL string `json:"imageUrl"`
	Prompt   string `json:"prompt"`
}

type SaveToGalleryRequest struct {
	ImageURL string `json:"imageUrl"`
	Prompt   string `json:"prompt"`
	Type     string `json:"type"` // "generated", "inpainted", "variation"
}

type ImageResponse struct {
	Success bool   `json:"success"`
	Image   *Image `json:"image,omitempty"`
	Error   string `json:"error,omitempty"`
}

// UploadImageRequest represents a request to upload an image directly to the gallery
type UploadImageRequest struct {
	Base64Image string `json:"base64Image" binding:"required"`
	Prompt      string `json:"prompt"`
}

// ReplicateAPIKey represents the user's Replicate API key stored in Firestore
type ReplicateAPIKey struct {
	Key string `json:"key" firestore:"key"`
}

// ReplicateAPIKeyRequest represents a request to set the Replicate API key
type ReplicateAPIKeyRequest struct {
	Key string `json:"key" binding:"required"`
}

// GalleryResponse represents a response with a list of images
type GalleryResponse struct {
	Success bool    `json:"success"`
	Images  []Image `json:"images,omitempty"`
	Error   string  `json:"error,omitempty"`
}
