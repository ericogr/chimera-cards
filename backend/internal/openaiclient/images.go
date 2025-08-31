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

// imagePromptTemplate can be set at application startup to customize the
// prompt used when requesting image generation from OpenAI. Use the token
// "{{animals}}" in the template where the comma-separated animal names
// should be substituted.
var imagePromptTemplate string

// SetImagePromptTemplate sets a custom prompt template for image
// generation. Call this during app initialization if you wish to override
// the built-in default.
func SetImagePromptTemplate(t string) {
	imagePromptTemplate = strings.TrimSpace(t)
}

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

	// Build prompt from template. The openaiclient package exposes a
	// configurable prompt template (see SetImagePromptTemplate). The
	// template should contain the token "{{animals}}" which will be
	// substituted with a comma-separated list of animal names. If no
	// custom template was provided, a sensible default is used.
	animalsPart := strings.Join(animalNames, ", ")
	prompt := imagePromptTemplate
	if prompt == "" {
		// default template that works for 1..3 animals
		prompt = "Create a single PNG image of {{animals}} in a comic-book superhero cartoon style. Vibrant colors, bold clean lines, dynamic heroic pose, no text or logos, transparent background. Combine distinctive features of each animal into a cohesive single creature."
	}
	prompt = strings.ReplaceAll(prompt, "{{animals}}", animalsPart)

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
