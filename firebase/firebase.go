package firebase

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

var (
	App           *firebase.App
	FirebaseAuth  *auth.Client
	firestoreClient *firestore.Client
)

// InitFirebase initializes the Firebase app and services
func InitFirebase() error {
	// Get the path to the service account key from environment variable or use default
	serviceAccountKeyPath := os.Getenv("FIREBASE_SERVICE_ACCOUNT_KEY")
	if serviceAccountKeyPath == "" {
		serviceAccountKeyPath = "serviceAccountKey.json"
	}

	// Initialize Firebase app with service account
	opt := option.WithCredentialsFile(serviceAccountKeyPath)
	var err error
	App, err = firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Error initializing Firebase app: %v", err)
		return err
	}

	// Initialize Firebase Auth
	FirebaseAuth, err = App.Auth(context.Background())
	if err != nil {
		log.Fatalf("Error initializing Firebase Auth: %v", err)
		return err
	}

	// Initialize Firestore
	firestoreClient, err = App.Firestore(context.Background())
	if err != nil {
		log.Fatalf("Error initializing Firestore: %v", err)
		return err
	}

	log.Println("Firebase initialized successfully")
	return nil
}

// GetFirestoreClient returns the Firestore client
func GetFirestoreClient() *firestore.Client {
	return firestoreClient
}

// GetAuthClient returns the Firebase Auth client
func GetAuthClient() *auth.Client {
	return FirebaseAuth
}
