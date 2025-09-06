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
	"github.com/ericogr/quimera-cards/internal/logging"
)

// Two prompt templates may be provided: one used when generating single
// animal portraits (singleImagePromptTemplate) and another used when
// generating hybrid images (hybridImagePromptTemplate). Each template may
// include the token "{{animals}}" which will be replaced by the comma-
// separated animal names.
var singleImagePromptTemplate string
var hybridImagePromptTemplate string

// SetSingleImagePromptTemplate sets the prompt template used when
// generating images for a single animal.
func SetSingleImagePromptTemplate(t string) {
	singleImagePromptTemplate = strings.TrimSpace(t)
}

// SetHybridImagePromptTemplate sets the prompt template used when
// generating images for hybrids composed of multiple animals.
func SetHybridImagePromptTemplate(t string) {
	hybridImagePromptTemplate = strings.TrimSpace(t)
}

// GenerateAnimalImage generates an image for a single animal using the
// configured animal prompt template (or a sensible default if missing).
func GenerateAnimalImage(ctx context.Context, animalName string) ([]byte, error) {
	if strings.TrimSpace(animalName) == "" {
		return nil, fmt.Errorf("animalName must be non-empty")
	}
	return generateImageWithTemplate(ctx, singleImagePromptTemplate, []string{animalName})
}

// GenerateHybridImage generates an image for a hybrid composed of 1..3
// animals using the configured hybrid prompt template (or a sensible
// default if missing).
func GenerateHybridImage(ctx context.Context, animalNames []string) ([]byte, error) {
	if len(animalNames) == 0 || len(animalNames) > 3 {
		return nil, fmt.Errorf("animalNames must contain 1..3 items")
	}
	return generateImageWithTemplate(ctx, hybridImagePromptTemplate, animalNames)
}

// generateImageWithTemplate is an internal helper that forms the prompt
// from the provided template (or a default) and calls the OpenAI API.
func generateImageWithTemplate(ctx context.Context, template string, animalNames []string) ([]byte, error) {
	if len(animalNames) == 0 || len(animalNames) > 3 {
		return nil, fmt.Errorf("animalNames must contain 1..3 items")
	}

	apiKey := os.Getenv(constants.EnvOpenAIAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("%s not set", constants.EnvOpenAIAPIKey)
	}

	animalsPart := strings.Join(animalNames, ", ")
	prompt := template
	if prompt == "" {
		// default hybrid-style template used when no custom template provided
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

	// Log the prompt before sending the request so operators can see what
	// was asked to the image API when a generation happens.
	logging.Info("openai image prompt", logging.Fields{"animals": animalsPart, "prompt": prompt})

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
