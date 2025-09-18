package openaiclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ericogr/chimera-cards/internal/constants"
	"github.com/ericogr/chimera-cards/internal/logging"
)

// Two prompt templates may be provided: one used when generating single
// entity portraits (singleImagePromptTemplate) and another used when
// generating hybrid images (hybridImagePromptTemplate). Each template may
// include the token "{{entities}}" which will be replaced by the comma-
// separated entity names.
var singleImagePromptTemplate string
var hybridImagePromptTemplate string

// SetSingleImagePromptTemplate sets the prompt template used when
// generating images for a single entity.
func SetSingleImagePromptTemplate(t string) {
	singleImagePromptTemplate = strings.TrimSpace(t)
}

// SetHybridImagePromptTemplate sets the prompt template used when
// generating images for hybrids composed of multiple entities.
func SetHybridImagePromptTemplate(t string) {
	hybridImagePromptTemplate = strings.TrimSpace(t)
}

// GenerateEntityImage generates an image for a single entity using the
// configured entity prompt template (or a sensible default if missing).
func GenerateEntityImage(ctx context.Context, entityName string) ([]byte, error) {
	if strings.TrimSpace(entityName) == "" {
		return nil, fmt.Errorf("entityName must be non-empty")
	}
	return generateImageWithTemplate(ctx, singleImagePromptTemplate, []string{entityName})
}

// GenerateHybridImage generates an image for a hybrid composed of 1..3
// entities using the configured hybrid prompt template (or a sensible
// default if missing).
func GenerateHybridImage(ctx context.Context, entityNames []string) ([]byte, error) {
	if len(entityNames) == 0 || len(entityNames) > 3 {
		return nil, fmt.Errorf("entityNames must contain 1..3 items")
	}
	return generateImageWithTemplate(ctx, hybridImagePromptTemplate, entityNames)
}

// generateImageWithTemplate is an internal helper that forms the prompt
// from the provided template (or a default) and calls the OpenAI API.
func generateImageWithTemplate(ctx context.Context, template string, entityNames []string) ([]byte, error) {
	if len(entityNames) == 0 || len(entityNames) > 3 {
		return nil, fmt.Errorf("entityNames must contain 1..3 items")
	}

	apiKey := os.Getenv(constants.EnvOpenAIAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%s not set", constants.EnvOpenAIAPIKey)
	}

	entitiesPart := strings.Join(entityNames, ", ")
	prompt := template
	if prompt == "" {
		// default hybrid-style template used when no custom template provided
		prompt = "Create a single PNG image of {{entities}} in a comic-book superhero cartoon style. Vibrant colors, bold clean lines, dynamic heroic pose, no text or logos, transparent background. Combine distinctive features of each entity into a cohesive single creature."
	}
	prompt = strings.ReplaceAll(prompt, "{{entities}}", entitiesPart)

	payload := map[string]interface{}{
		"prompt":  prompt,
		"n":       1,
		"size":    constants.OpenAIImageSizeDefault,
		"model":   constants.OpenAIImageModel,
		"quality": constants.OpenAIImageQualityDefault,
	}

	// Log the prompt before sending the request so operators can see what
	// was asked to the image API when a generation happens.
	logging.Info("openai image prompt", logging.Fields{"entities": entitiesPart, "prompt": prompt})

	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", constants.OpenAIBaseURL+constants.OpenAIImagesGenerationsPath, strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	req.Header.Set(constants.HeaderAuthorization, constants.BearerPrefix+apiKey)
	req.Header.Set(constants.HeaderContentType, constants.ContentTypeJSON)

	// Try up to N attempts in case the OpenAI image endpoint transiently
	// rejects requests. Use exponential backoff between attempts. If the
	// provided context is canceled, abort early.
	const maxAttempts = 3
	client := &http.Client{Timeout: 60 * time.Second}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Recreate request body reader for each attempt
		req, err := http.NewRequestWithContext(ctx, "POST", constants.OpenAIBaseURL+constants.OpenAIImagesGenerationsPath, strings.NewReader(string(b)))
		if err != nil {
			return nil, err
		}
		req.Header.Set(constants.HeaderAuthorization, constants.BearerPrefix+apiKey)
		req.Header.Set(constants.HeaderContentType, constants.ContentTypeJSON)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			logging.Error("openai image request failed", err, logging.Fields{"attempt": attempt, "entities": entitiesPart})
		} else {
			// Ensure body closed for this attempt
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				lastErr = fmt.Errorf("openai image generation failed: %d %s", resp.StatusCode, string(bodyBytes))
				logging.Error("openai image generation failed", lastErr, logging.Fields{"attempt": attempt, "entities": entitiesPart})
			} else {
				var out struct {
					Data []struct {
						B64JSON string `json:"b64_json"`
						URL     string `json:"url"`
					} `json:"data"`
				}
				if err := json.Unmarshal(bodyBytes, &out); err != nil {
					lastErr = fmt.Errorf("failed to decode OpenAI response: %w", err)
					logging.Error("openai image decode failed", lastErr, logging.Fields{"attempt": attempt, "entities": entitiesPart})
				} else if len(out.Data) == 0 {
					lastErr = fmt.Errorf("openai returned no image data")
					logging.Error("openai returned no image data", lastErr, logging.Fields{"attempt": attempt, "entities": entitiesPart})
				} else if out.Data[0].B64JSON != "" {
					imgBytes, err := base64.StdEncoding.DecodeString(out.Data[0].B64JSON)
					if err != nil {
						lastErr = fmt.Errorf("failed to decode base64 image: %w", err)
						logging.Error("openai image base64 decode failed", lastErr, logging.Fields{"attempt": attempt, "entities": entitiesPart})
					} else {
						return imgBytes, nil
					}
				} else {
					lastErr = fmt.Errorf("openai returned unsupported image payload")
					logging.Error("openai returned unsupported image payload", lastErr, logging.Fields{"attempt": attempt, "entities": entitiesPart})
				}
			}
		}

		// If context cancelled, abort early
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Backoff before next attempt
		if attempt < maxAttempts {
			backoff := time.Duration(attempt) * 500 * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			logging.Info("retrying openai image generation", logging.Fields{"attempt": attempt + 1, "entities": entitiesPart})
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("openai image generation failed")
}
