package controllers

import (
	"backend/models"
	"context"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

type AvatarController struct {
	firestoreClient *firestore.Client
}

func NewAvatarController(firestoreClient *firestore.Client) *AvatarController {
	return &AvatarController{
		firestoreClient: firestoreClient,
	}
}

// CreateAvatar creates a new avatar
func (ac *AvatarController) CreateAvatar(c *gin.Context) {
	var req models.AvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userId")
	if userID == "" {
		log.Printf("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	now := time.Now().Unix()
	avatar := models.Avatar{
		ID:              uuid.New().String(),
		Name:            req.Name,
		Description:     req.Description,
		Story:           req.Story,
		Persona:         req.Persona,
		ProfileImageURL: req.ProfileImageURL,
		IsPublic:        req.IsPublic,
		OwnerID:         userID,
		CreatorNickname: req.CreatorNickname,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err := ac.firestoreClient.Collection("avatars").Doc(avatar.ID).Set(context.Background(), avatar)
	if err != nil {
		log.Printf("Error creating avatar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create avatar"})
		return
	}

	c.JSON(http.StatusCreated, avatar)
}

// GetAvatar gets a single avatar by ID
func (ac *AvatarController) GetAvatar(c *gin.Context) {
	avatarID := c.Param("id")
	if avatarID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Avatar ID is required"})
		return
	}

	userID := c.GetString("userId")
	if userID == "" {
		log.Printf("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Get the avatar from Firestore
	avatarDoc, err := ac.firestoreClient.Collection("avatars").Doc(avatarID).Get(context.Background())
	if err != nil {
		log.Printf("Error getting avatar: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Avatar not found"})
		return
	}

	var avatar models.Avatar
	if err := avatarDoc.DataTo(&avatar); err != nil {
		log.Printf("Error converting document to avatar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process avatar data"})
		return
	}

	// Check if the avatar is public or belongs to the user
	if !avatar.IsPublic && avatar.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have access to this avatar"})
		return
	}

	// If creator nickname is empty, try to get the user's display name from Firestore
	if avatar.CreatorNickname == "" {
		// Get the user document from Firestore
		userDoc, err := ac.firestoreClient.Collection("users").Doc(avatar.OwnerID).Get(context.Background())
		if err == nil {
			// If we found the user document, extract the display name
			userData := map[string]interface{}{}
			if err := userDoc.DataTo(&userData); err == nil {
				if displayName, ok := userData["displayName"].(string); ok && displayName != "" {
					avatar.CreatorNickname = displayName
				}
			}
		}
		
		// If we still don't have a nickname, use a default
		if avatar.CreatorNickname == "" {
			avatar.CreatorNickname = "Anonymous Creator"
		}
	}

	// Return the avatar with ownership information
	avatarResponse := models.AvatarResponse{
		Avatar:  avatar,
		IsOwner: avatar.OwnerID == userID,
	}

	c.JSON(http.StatusOK, avatarResponse)
}

// GetUserAvatars gets all avatars owned by the current user
func (ac *AvatarController) GetUserAvatars(c *gin.Context) {
	userID := c.GetString("userId")
	if userID == "" {
		log.Printf("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	iter := ac.firestoreClient.Collection("avatars").Where("ownerId", "==", userID).Documents(context.Background())
	ac.getAvatarsFromIterator(c, iter, userID)
}

// GetPublicAvatars gets all public avatars for the marketplace
func (ac *AvatarController) GetPublicAvatars(c *gin.Context) {
	userID := c.GetString("userId")
	// For public avatars, we allow access even without authentication
	// but we still use the userID if available to mark owned avatars

	iter := ac.firestoreClient.Collection("avatars").Where("isPublic", "==", true).Documents(context.Background())
	
	var avatars []models.AvatarResponse
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating avatars: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve avatars"})
			return
		}

		var avatar models.Avatar
		if err := doc.DataTo(&avatar); err != nil {
			log.Printf("Error converting document to avatar: %v", err)
			continue
		}

		avatarResponse := models.AvatarResponse{
			Avatar:  avatar,
			IsOwner: userID != "" && avatar.OwnerID == userID,
		}
		avatars = append(avatars, avatarResponse)
	}

	c.JSON(http.StatusOK, gin.H{"avatars": avatars})
}

// Helper function to process avatar iterators
func (ac *AvatarController) getAvatarsFromIterator(c *gin.Context, iter *firestore.DocumentIterator, userID string) {
	var avatars []models.AvatarResponse
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error iterating avatars: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve avatars"})
			return
		}

		var avatar models.Avatar
		if err := doc.DataTo(&avatar); err != nil {
			log.Printf("Error converting document to avatar: %v", err)
			continue
		}

		avatarResponse := models.AvatarResponse{
			Avatar:  avatar,
			IsOwner: avatar.OwnerID == userID,
		}
		avatars = append(avatars, avatarResponse)
	}

	c.JSON(http.StatusOK, gin.H{"avatars": avatars})
}

// UpdateAvatar updates an existing avatar
func (ac *AvatarController) UpdateAvatar(c *gin.Context) {
	avatarID := c.Param("id")
	if avatarID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Avatar ID is required"})
		return
	}

	var req models.AvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userId")
	if userID == "" {
		log.Printf("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Check if the avatar exists and belongs to the user
	doc, err := ac.firestoreClient.Collection("avatars").Doc(avatarID).Get(context.Background())
	if err != nil {
		log.Printf("Error getting avatar: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Avatar not found"})
		return
	}

	var existingAvatar models.Avatar
	if err := doc.DataTo(&existingAvatar); err != nil {
		log.Printf("Error converting document to avatar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process avatar data"})
		return
	}

	if existingAvatar.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this avatar"})
		return
	}

	// Update the avatar
	updates := []firestore.Update{
		{Path: "name", Value: req.Name},
		{Path: "description", Value: req.Description},
		{Path: "story", Value: req.Story},
		{Path: "persona", Value: req.Persona},
		{Path: "profileImageUrl", Value: req.ProfileImageURL},
		{Path: "isPublic", Value: req.IsPublic},
		{Path: "updatedAt", Value: time.Now().Unix()},
	}
	
	// Only update creatorNickname if it's provided
	if req.CreatorNickname != "" {
		updates = append(updates, firestore.Update{Path: "creatorNickname", Value: req.CreatorNickname})
	}

	_, err = ac.firestoreClient.Collection("avatars").Doc(avatarID).Update(context.Background(), updates)
	if err != nil {
		log.Printf("Error updating avatar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update avatar"})
		return
	}

	// Get the updated avatar
	updatedDoc, err := ac.firestoreClient.Collection("avatars").Doc(avatarID).Get(context.Background())
	if err != nil {
		log.Printf("Error getting updated avatar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve updated avatar"})
		return
	}

	var updatedAvatar models.Avatar
	if err := updatedDoc.DataTo(&updatedAvatar); err != nil {
		log.Printf("Error converting document to avatar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process updated avatar data"})
		return
	}

	response := models.AvatarResponse{
		Avatar:  updatedAvatar,
		IsOwner: true,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteAvatar deletes an avatar
func (ac *AvatarController) DeleteAvatar(c *gin.Context) {
	avatarID := c.Param("id")
	if avatarID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Avatar ID is required"})
		return
	}

	userID := c.GetString("userId")
	if userID == "" {
		log.Printf("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Check if the avatar exists and belongs to the user
	doc, err := ac.firestoreClient.Collection("avatars").Doc(avatarID).Get(context.Background())
	if err != nil {
		log.Printf("Error getting avatar: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Avatar not found"})
		return
	}

	var existingAvatar models.Avatar
	if err := doc.DataTo(&existingAvatar); err != nil {
		log.Printf("Error converting document to avatar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process avatar data"})
		return
	}

	if existingAvatar.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this avatar"})
		return
	}

	// Delete the avatar
	_, err = ac.firestoreClient.Collection("avatars").Doc(avatarID).Delete(context.Background())
	if err != nil {
		log.Printf("Error deleting avatar: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete avatar"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Avatar deleted successfully"})
}
