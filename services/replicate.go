package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type ReplicateService struct {
	apiKey     string
	httpClient *http.Client
}

type ReplicateRequest struct {
	Version string                 `json:"version"`
	Input   map[string]interface{} `json:"input"`
}

// Use interface{} for status and output to handle different field types
type ReplicateResponse struct {
	ID     string      `json:"id"`
	Status interface{} `json:"status"`
	Output interface{} `json:"output"` // Changed from []string to interface{} to handle both string and array outputs
	Error  string      `json:"error"`
}

func NewReplicateService() *ReplicateService {
	apiKey := os.Getenv("REPLICATE_API_KEY")

	// Debug log to check if API key is loaded
	if apiKey == "" {
		log.Printf("WARNING: REPLICATE_API_KEY environment variable is empty")
		// Return a service that will return errors when used
		return &ReplicateService{
			apiKey: "",
			httpClient: &http.Client{
				Timeout: time.Minute * 5, // Increase timeout to 5 minutes
			},
		}
	} else {
		// Only log first few characters for security
		if len(apiKey) > 10 {
			log.Printf("Replicate API key loaded: %s...", apiKey[:10])
		} else {
			log.Printf("Replicate API key loaded but seems too short")
		}
	}

	// Ensure the API key is properly formatted (should start with r8_)
	if !strings.HasPrefix(apiKey, "r8_") {
		log.Printf("WARNING: Replicate API key does not start with 'r8_', which is the expected format")
	}

	return &ReplicateService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: time.Minute * 5, // Increase timeout to 5 minutes
		},
	}
}

// NewReplicateServiceWithKey creates a new ReplicateService with the provided API key
func NewReplicateServiceWithKey(apiKey string) *ReplicateService {
	return &ReplicateService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: time.Minute * 5,
		},
	}
}

// ValidateAPIKey checks if the provided API key is valid
func (s *ReplicateService) ValidateAPIKey() error {
	// Check if API key is empty
	if s.apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Check if key has the expected format
	if !strings.HasPrefix(s.apiKey, "r8_") {
		return fmt.Errorf("Invalid API key format. Replicate API keys should start with 'r8_'")
	}

	// Make a lightweight API call to verify the key
	req, err := http.NewRequest("GET", "https://api.replicate.com/v1/models", nil)
	if err != nil {
		return fmt.Errorf("Error creating request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", s.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("Invalid API key")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned error status: %d", resp.StatusCode)
	}

	return nil
}

func (s *ReplicateService) GenerateImage(prompt string) (string, error) {
	// don't touch this model
	modelVersion := "black-forest-labs/flux-schnell"

	input := map[string]interface{}{
		"prompt":      prompt,
		"width":       512,
		"height":      512,
		"num_outputs": 1,
	}

	log.Printf("Generating image with model: %s", modelVersion)
	log.Printf("Input parameters: %+v", input)

	return s.predict(modelVersion, input)
}

func (s *ReplicateService) InpaintImage(imageURL, prompt, maskURL string) (string, error) {
	// Using the working model from the provided code
	modelVersion := "zsxkib/flux-dev-inpainting:11cca3274341de7aef06f04e4dab3d651ea8ac04eff003f23603d4fdf5b56ff0"

	// Match the input parameters from the working code
	input := map[string]interface{}{
		"image":       imageURL,
		"mask":        maskURL,
		"prompt":      prompt,
		"num_outputs": 1,
	}

	log.Printf("Inpainting with model: %s", modelVersion)
	log.Printf("Input parameters: %+v", input)

	return s.predict(modelVersion, input)
}

// Helper function to get status as string
func getStatusString(status interface{}) string {
	switch v := status.(type) {
	case string:
		return v
	case float64:
		if v == 1 {
			return "succeeded"
		} else if v == 0 {
			return "failed"
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (s *ReplicateService) predict(version string, input map[string]interface{}) (string, error) {
	// Check if API key is empty
	if s.apiKey == "" {
		return "", fmt.Errorf("Replicate API key is not set")
	}

	// Split version into model and version parts
	parts := strings.Split(version, ":")
	var modelName, versionID string
	var useDirectModelEndpoint bool

	if len(parts) == 2 {
		// Format: "owner/model:version"
		modelName = parts[0]
		versionID = parts[1]
	} else if len(parts) == 1 {
		// Format: "owner/model" - might need to use direct model endpoint
		modelName = parts[0]

		// Special case for models that need direct model endpoint
		if modelName == "black-forest-labs/flux-schnell" {
			useDirectModelEndpoint = true
			// No version needed for direct model endpoint
		} else {
			// For other models without explicit versions, we need a version
			return "", fmt.Errorf("model without version not supported: %s", modelName)
		}
	} else {
		return "", fmt.Errorf("invalid model version format: %s", version)
	}

	// Log the model and version for debugging
	if useDirectModelEndpoint {
		log.Printf("Using direct model endpoint for: %s", modelName)
	} else {
		log.Printf("Using model: %s, version: %s", modelName, versionID)
	}

	var jsonData []byte
	var err error
	var apiURL string

	if useDirectModelEndpoint {
		// For direct model endpoint, we need to wrap the input in an "input" object
		requestBody := map[string]interface{}{
			"input": input,
		}

		jsonData, err = json.Marshal(requestBody)
		if err != nil {
			return "", fmt.Errorf("error marshaling request: %v", err)
		}

		// Use the direct model endpoint
		apiURL = fmt.Sprintf("https://api.replicate.com/v1/models/%s/predictions", modelName)
	} else {
		// Standard endpoint with version
		reqBody := ReplicateRequest{
			Version: versionID,
			Input:   input,
		}

		jsonData, err = json.Marshal(reqBody)
		if err != nil {
			return "", fmt.Errorf("error marshaling request: %v", err)
		}

		// Use the standard predictions endpoint
		apiURL = "https://api.replicate.com/v1/predictions"
	}

	// Log the request body and URL for debugging
	log.Printf("Request URL: %s", apiURL)
	log.Printf("Request body: %s", string(jsonData))

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Debug log to check the authorization header
	authHeader := fmt.Sprintf("Token %s", s.apiKey)
	if len(s.apiKey) > 10 {
		log.Printf("Using Authorization header: Token %s...", s.apiKey[:10]) // Only log first few characters for security
	} else {
		log.Printf("WARNING: API key is too short, authentication will likely fail")
	}

	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	// Log the raw response for debugging
	log.Printf("Replicate API response status: %d", resp.StatusCode)
	log.Printf("Replicate API response body: %s", string(bodyBytes))

	// Check if the response status code is not successful
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("API returned error status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Create a new reader with the same data for JSON decoding
	respBodyReader := bytes.NewReader(bodyBytes)

	var prediction ReplicateResponse
	if err := json.NewDecoder(respBodyReader).Decode(&prediction); err != nil {
		return "", fmt.Errorf("error decoding response: %v, body: %s", err, string(bodyBytes))
	}

	// Log the parsed response
	log.Printf("Parsed response - ID: %s, Status: %v, Error: %s",
		prediction.ID, prediction.Status, prediction.Error)

	if prediction.ID == "" {
		return "", fmt.Errorf("no prediction ID returned from API, body: %s", string(bodyBytes))
	}

	// Poll for completion
	maxRetries := 60                // Increase max retries
	pollInterval := 3 * time.Second // Increase poll interval

	for i := 0; i < maxRetries; i++ {
		statusStr := getStatusString(prediction.Status)
		log.Printf("Current status: %s (original: %v)", statusStr, prediction.Status)

		if statusStr == "succeeded" {
			// Handle the output based on its type
			if prediction.Output != nil {
				log.Printf("Prediction succeeded with output type: %T, value: %v", prediction.Output, prediction.Output)

				// Handle different output types
				switch output := prediction.Output.(type) {
				case []interface{}:
					if len(output) > 0 {
						// Convert the first element to string
						if str, ok := output[0].(string); ok {
							return str, nil
						}
					}
				case string:
					// If output is directly a string
					return output, nil
				}

				// If we couldn't extract a string, return the raw output as JSON
				outputJSON, _ := json.Marshal(prediction.Output)
				return string(outputJSON), nil
			}

			return "", fmt.Errorf("prediction succeeded but no output was returned")
		}

		if statusStr == "failed" {
			return "", fmt.Errorf("prediction failed: %s", prediction.Error)
		}

		log.Printf("Waiting for prediction to complete, attempt %d/%d", i+1, maxRetries)
		time.Sleep(pollInterval)

		pollURL := fmt.Sprintf("https://api.replicate.com/v1/predictions/%s", prediction.ID)
		req, err = http.NewRequest("GET", pollURL, nil)
		if err != nil {
			return "", fmt.Errorf("error creating poll request: %v", err)
		}

		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")

		resp, err = s.httpClient.Do(req)
		if err != nil {
			log.Printf("Error polling prediction (will retry): %v", err)
			continue
		}

		// Read the poll response body for debugging
		pollBodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			return "", fmt.Errorf("error reading poll response body: %v", err)
		}

		// Log the raw poll response for debugging
		log.Printf("Replicate API poll response status: %d", resp.StatusCode)
		log.Printf("Replicate API poll response body: %s", string(pollBodyBytes))

		// Check if the poll response status code is not successful
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			log.Printf("API returned error status %d during polling: %s", resp.StatusCode, string(pollBodyBytes))
			continue
		}

		// Create a new reader with the same data for JSON decoding
		pollBodyReader := bytes.NewReader(pollBodyBytes)

		// Reset the prediction object before decoding
		prediction = ReplicateResponse{}

		if err := json.NewDecoder(pollBodyReader).Decode(&prediction); err != nil {
			resp.Body.Close()
			log.Printf("Error decoding poll response (will retry): %v, body: %s", err, string(pollBodyBytes))
			continue
		}
		resp.Body.Close()
	}

	return "", fmt.Errorf("timeout waiting for prediction after %d attempts", maxRetries)
}
