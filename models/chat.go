package models

type Chat struct {
	ID        string    `json:"id" firestore:"id"`
	UserID    string    `json:"userId" firestore:"userId"`
	Title     string    `json:"title" firestore:"title"`
	CreatedAt int64     `json:"createdAt" firestore:"createdAt"`
	UpdatedAt int64     `json:"updatedAt" firestore:"updatedAt"`
	Messages  []Message `json:"messages" firestore:"messages"`
	ModelID   string    `json:"modelId" firestore:"modelId"`
	AvatarID  string    `json:"avatarId" firestore:"avatarId"`             // Keep for backward compatibility
	AvatarIDs []string  `json:"avatarIds" firestore:"avatarIds,omitempty"` // New field for multiple avatars
}

type Message struct {
	Role      string `json:"role" firestore:"role"`
	Content   string `json:"content" firestore:"content"`
	Timestamp int64  `json:"timestamp" firestore:"timestamp"`
}

type ChatRequest struct {
	Message   string   `json:"message"`
	ModelID   string   `json:"modelId" binding:"required"`
	AvatarID  string   `json:"avatarId"`            // Keep for backward compatibility
	AvatarIDs []string `json:"avatarIds,omitempty"` // New field for multiple avatars
}

type OpenRouterAPIKey struct {
	Key string `json:"key" firestore:"key"`
}

type OpenRouterModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Context       int    `json:"context"`
	PricePerToken struct {
		Prompt     float64 `json:"prompt"`
		Completion float64 `json:"completion"`
	} `json:"pricePerToken"`
}

type APIKeyRequest struct {
	Key string `json:"key" binding:"required"`
}
