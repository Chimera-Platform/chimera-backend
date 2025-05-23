package services

import (
	"backend/models"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

const (
	OPENROUTER_API_URL     = "https://openrouter.ai/api/v1/chat/completions"
	OPENROUTER_MODELS_URL  = "https://openrouter.ai/api/v1/models"
	OPENROUTER_CREDITS_URL = "https://openrouter.ai/api/v1/credits"
)

type OpenRouterService struct {
	APIKey string
}

type OpenRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterRequest struct {
	Model    string              `json:"model"`
	Messages []OpenRouterMessage `json:"messages"`
	Stream   bool                `json:"stream,omitempty"`
}

type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type OpenRouterStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		Index int `json:"index"`
	} `json:"choices"`
}

type OpenRouterModelResponse struct {
	Data []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Created      int64  `json:"created"`
		Description  string `json:"description"`
		Context      int    `json:"context_length"`
		Architecture struct {
			Modality     string `json:"modality"`
			Tokenizer    string `json:"tokenizer"`
			InstructType any    `json:"instruct_type"`
		} `json:"architecture"`
		Pricing struct {
			Prompt     string `json:"prompt"`
			Completion string `json:"completion"`
			Image      string `json:"image"`
			Request    string `json:"request"`
		} `json:"pricing"`
		TopProvider struct {
			ContextLength       int  `json:"context_length"`
			MaxCompletionTokens any  `json:"max_completion_tokens"`
			IsModerated         bool `json:"is_moderated"`
		} `json:"top_provider"`
		PerRequestLimits any `json:"per_request_limits"`
	} `json:"data"`
}

// OpenRouterCredits represents the response from the credits endpoint
type OpenRouterCredits struct {
	Data struct {
		TotalCredits float64 `json:"total_credits"`
		TotalUsage   float64 `json:"total_usage"`
	} `json:"data"`
}

func NewOpenRouterService(apiKey string) *OpenRouterService {
	return &OpenRouterService{
		APIKey: apiKey,
	}
}

func (s *OpenRouterService) SendMessage(modelID string, messages []OpenRouterMessage) (string, error) {
	if s.APIKey == "" {
		return "", errors.New("API key not set")
	}

	reqBody := OpenRouterRequest{
		Model:    modelID,
		Messages: messages,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("POST", OPENROUTER_API_URL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.APIKey))
	req.Header.Set("HTTP-Referer", "https://github.com/") // Required by OpenRouter
	req.Header.Set("X-Title", "Chat Application")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Try to decode the response
	var openRouterResp OpenRouterResponse
	if err := json.Unmarshal(bodyBytes, &openRouterResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %v\nResponse body: %s", err, string(bodyBytes))
	}

	if len(openRouterResp.Choices) == 0 {
		return "", fmt.Errorf("no response from the model\nResponse body: %s", string(bodyBytes))
	}

	return openRouterResp.Choices[0].Message.Content, nil
}

// GetModels fetches available models from OpenRouter
func (s *OpenRouterService) GetModels() ([]models.OpenRouterModel, error) {
	if s.APIKey == "" {
		return nil, errors.New("API key not set")
	}

	req, err := http.NewRequest("GET", OPENROUTER_MODELS_URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.APIKey))
	req.Header.Set("HTTP-Referer", "https://github.com/") // Required by OpenRouter
	req.Header.Set("X-Title", "Chat Application")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Try to decode the response
	var modelResp OpenRouterModelResponse
	if err := json.Unmarshal(bodyBytes, &modelResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v\nResponse body: %s", err, string(bodyBytes))
	}

	var result []models.OpenRouterModel
	for _, m := range modelResp.Data {
		// Convert string pricing to float64
		promptPrice, err := strconv.ParseFloat(m.Pricing.Prompt, 64)
		if err != nil {
			promptPrice = 0
		}
		completionPrice, err := strconv.ParseFloat(m.Pricing.Completion, 64)
		if err != nil {
			completionPrice = 0
		}

		model := models.OpenRouterModel{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Context:     m.Context,
		}
		model.PricePerToken.Prompt = promptPrice
		model.PricePerToken.Completion = completionPrice
		result = append(result, model)
	}

	return result, nil
}

// ValidateAPIKey checks if the API key is valid by attempting to fetch models
func (s *OpenRouterService) ValidateAPIKey() error {
	req, err := http.NewRequest("GET", OPENROUTER_MODELS_URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.APIKey))
	req.Header.Set("HTTP-Referer", "https://github.com/") // Required by OpenRouter
	req.Header.Set("X-Title", "Chat Application")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API validation failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// GetCredits fetches the user's credit balance from OpenRouter
func (s *OpenRouterService) GetCredits() (*OpenRouterCredits, error) {
	if s.APIKey == "" {
		return nil, errors.New("API key not set")
	}

	req, err := http.NewRequest("GET", OPENROUTER_CREDITS_URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.APIKey))
	req.Header.Set("HTTP-Referer", "https://github.com/") // Required by OpenRouter
	req.Header.Set("X-Title", "Chat Application")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Try to decode the response
	var credits OpenRouterCredits
	if err := json.Unmarshal(bodyBytes, &credits); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v\nResponse body: %s", err, string(bodyBytes))
	}

	return &credits, nil
}

// SendMessageStream sends a message to OpenRouter and streams the response
func (s *OpenRouterService) SendMessageStream(modelID string, messages []OpenRouterMessage, writer http.ResponseWriter) error {
	if s.APIKey == "" {
		return errors.New("API key not set")
	}

	reqBody := OpenRouterRequest{
		Model:    modelID,
		Messages: messages,
		Stream:   true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("POST", OPENROUTER_API_URL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.APIKey))
	req.Header.Set("HTTP-Referer", "https://github.com/") // Required by OpenRouter
	req.Header.Set("X-Title", "Chat Application")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Create a flusher to ensure data is sent immediately
	flusher, ok := writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	// Process the streaming response
	scanner := bufio.NewScanner(resp.Body)
	fullContent := ""

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// SSE format starts with "data: "
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		// Remove the "data: " prefix
		data := strings.TrimPrefix(line, "data: ")

		// Check for the end of the stream
		if data == "[DONE]" {
			// Send a final message to indicate the end of the stream
			fmt.Fprintf(writer, "data: [DONE]\n\n")
			flusher.Flush()
			break
		}

		// Parse the JSON data
		var streamResp OpenRouterStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			// If we can't parse the JSON, just send the raw data
			fmt.Fprintf(writer, "data: %s\n\n", data)
			flusher.Flush()
			continue
		}

		// Extract the content
		if len(streamResp.Choices) > 0 {
			content := streamResp.Choices[0].Delta.Content
			if content != "" {
				fullContent += content

				// Send the chunk to the client
				fmt.Fprintf(writer, "data: %s\n\n", content)
				flusher.Flush()
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %v", err)
	}

	return nil
}
