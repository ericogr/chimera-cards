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

	"github.com/ericogr/quimera-cards/internal/constants"
)

// GenerateImageFromNames calls the OpenAI Images API to generate a single PNG
// image (256/1024 depending on constants) for the provided animal names.
// It returns the raw image bytes (PNG) or an error.
func GenerateImageFromNames(ctx context.Context, animalNames []string) ([]byte, error) {
	if len(animalNames) == 0 || len(animalNames) > 3 {
		return nil, fmt.Errorf("animalNames must contain 1..3 items")
	}

	apiKey := os.Getenv(constants.EnvOpenAIAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%s not set", constants.EnvOpenAIAPIKey)
	}

	// Build prompt
	var prompt string
	if len(animalNames) == 1 {
		prompt = fmt.Sprintf("Create a single %s PNG image of a %s in a comic-book superhero cartoon style. Vibrant colors, bold clean lines, dynamic heroic pose, no text or logos, transparent background.", constants.OpenAIImageSizeDefault, animalNames[0])
	} else {
		prompt = fmt.Sprintf("Create a single %s PNG image of a hybrid creature that merges these animals into one chimeric superhero: %s. Style: comic-book superhero cartoon, vibrant colors, bold clean lines, dynamic heroic pose, no text or logos, transparent background. Combine distinctive features of each animal into a cohesive single creature.", constants.OpenAIImageSizeDefault, strings.Join(animalNames, ", "))
	}

	payload := map[string]interface{}{
		"prompt":  prompt,
		"n":       1,
		"size":    constants.OpenAIImageSizeDefault,
		"model":   constants.OpenAIImageModel,
		"quality": constants.OpenAIImageQualityDefault,
	}

	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", constants.OpenAIBaseURL+constants.OpenAIImagesGenerationsPath, strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	req.Header.Set(constants.HeaderAuthorization, constants.BearerPrefix+apiKey)
	req.Header.Set(constants.HeaderContentType, constants.ContentTypeJSON)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai image generation failed: %d %s", resp.StatusCode, string(body))
	}

	var out struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI response: %w", err)
	}
	if len(out.Data) == 0 {
		return nil, fmt.Errorf("openai returned no image data")
	}

	if out.Data[0].B64JSON != "" {
		imgBytes, err := base64.StdEncoding.DecodeString(out.Data[0].B64JSON)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 image: %w", err)
		}
		return imgBytes, nil
	}

	return nil, fmt.Errorf("openai returned unsupported image payload")
}
