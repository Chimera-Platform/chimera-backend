package models

// Avatar represents a character that can be used in chats
type Avatar struct {
	ID              string `json:"id" firestore:"id"`
	Name            string `json:"name" firestore:"name"`
	Description     string `json:"description" firestore:"description"`
	Story           string `json:"story" firestore:"story"`
	Persona         string `json:"persona" firestore:"persona"`
	ProfileImageURL string `json:"profileImageUrl" firestore:"profileImageUrl"`
	IsPublic        bool   `json:"isPublic" firestore:"isPublic"`
	OwnerID         string `json:"ownerId" firestore:"ownerId"`
	CreatorNickname string `json:"creatorNickname" firestore:"creatorNickname"`
	CreatedAt       int64  `json:"createdAt" firestore:"createdAt"`
	UpdatedAt       int64  `json:"updatedAt" firestore:"updatedAt"`
}

// AvatarRequest is used for creating or updating an avatar
type AvatarRequest struct {
	Name            string `json:"name" binding:"required"`
	Description     string `json:"description" binding:"required"`
	Story           string `json:"story" binding:"required"`
	Persona         string `json:"persona" binding:"required"`
	ProfileImageURL string `json:"profileImageUrl" binding:"required"`
	IsPublic        bool   `json:"isPublic"`
	CreatorNickname string `json:"creatorNickname"`
}

// AvatarResponse is used for returning avatar data with additional metadata
type AvatarResponse struct {
	Avatar
	IsOwner bool `json:"isOwner"`
} 