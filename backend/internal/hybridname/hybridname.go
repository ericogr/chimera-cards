package hybridname

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ericogr/quimera-cards/internal/constants"
	"github.com/ericogr/quimera-cards/internal/dedupe"
	"github.com/ericogr/quimera-cards/internal/logging"

	"github.com/ericogr/quimera-cards/internal/storage"
)

// buildKeyFromIDs returns a canonical key for a list of animal IDs, e.g. "1,3,7".
func buildKeyFromIDs(ids []uint) string {
	if len(ids) == 0 {
		return ""
	}
	ints := make([]int, len(ids))
	for i, v := range ids {
		ints[i] = int(v)
	}
	sort.Ints(ints)
	parts := make([]string, len(ints))
	for i, v := range ints {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

// callOpenAI invokes the OpenAI Chat Completions API to generate a single
// creative name for the provided animal names. It returns the generated name
// or an error if the request failed.
func callOpenAI(animalNames []string) (string, error) {
	apiKey := os.Getenv(constants.EnvOpenAIAPIKey)
	if apiKey == "" {
		return "", fmt.Errorf("%s not set", constants.EnvOpenAIAPIKey)
	}

	prompt := fmt.Sprintf("Given these animal names: %s. Create a short, fun, single-name hybrid that combines them (1-3 words). Return only the name.", strings.Join(animalNames, ", "))

	payload := map[string]interface{}{
		"model": constants.OpenAIChatModel,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a creative name generator for game creatures."},
			{"role": "user", "content": prompt},
		},
		"max_completion_tokens": 3100,
		"service_tier":          "default", //flex
	}

	b, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", constants.OpenAIBaseURL+constants.OpenAIChatCompletionsPath, bytes.NewBuffer(b))
	if err != nil {
		return "", err
	}
	req.Header.Set(constants.HeaderAuthorization, constants.BearerPrefix+apiKey)
	req.Header.Set(constants.HeaderContentType, constants.ContentTypeJSON)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai error: %d %s", resp.StatusCode, string(body))
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenAI")
	}

	name := strings.TrimSpace(out.Choices[0].Message.Content)
	// Take only the first line and trim surrounding quotes/spaces
	if idx := strings.Index(name, "\n"); idx >= 0 {
		name = name[:idx]
	}
	name = strings.Trim(name, "\"' ")
	return name, nil
}

// GetOrCreateGeneratedName checks the repository for an existing generated name
// for the given animal IDs; if not found, it calls OpenAI to generate one and
// stores it in the repository. It returns the name, the source ("db"|"openai"),
// and an error if the OpenAI call failed.
func GetOrCreateGeneratedName(repo storage.Repository, animalIDs []uint, animalNames []string) (string, string, error) {
	// Build canonical animal key from names: lowercase, underscores, sorted.
	parts := make([]string, 0, len(animalNames))
	for _, p := range animalNames {
		q := strings.TrimSpace(p)
		if q != "" {
			parts = append(parts, strings.ToLower(strings.ReplaceAll(q, " ", "_")))
		}
	}
	sort.Strings(parts)
	animalKey := strings.Join(parts, "_")

	// Try cache by canonical name-key first.
	if animalKey != "" {
		if gn, err := repo.GetGeneratedNameByAnimalKey(animalKey); err == nil && gn != nil && gn.GeneratedName != "" {
			logging.Info("hybrid-name cache hit by animal_key", logging.Fields{constants.LogFieldKey: animalKey, constants.LogFieldName: gn.GeneratedName, constants.LogFieldSource: "db_key"})
			return gn.GeneratedName, "db_key", nil
		}
	}

	// Not cached — deduplicate concurrent generation using singleflight
	// keyed by the canonical animalKey (fallback to a stable string if
	// animalKey is empty).
	sfKey := animalKey
	if sfKey == "" {
		// As a last resort use the joined animal names string (unsorted)
		sfKey = strings.Join(animalNames, " + ")
	}

	type genRes struct {
		Name   string
		Source string
	}

	ch := dedupe.NameGroup.DoChan(sfKey, func() (interface{}, error) {
		// Re-check DB by animal key inside the singleflight function in
		// case another goroutine saved the generated name before we got here.
		if animalKey != "" {
			if gn, err := repo.GetGeneratedNameByAnimalKey(animalKey); err == nil && gn != nil && gn.GeneratedName != "" {
				logging.Info("hybrid-name cache hit (singleflight)", logging.Fields{constants.LogFieldKey: animalKey, constants.LogFieldName: gn.GeneratedName, constants.LogFieldSource: "db_key"})
				return genRes{Name: gn.GeneratedName, Source: "db_key"}, nil
			}
		}

		// Ask OpenAI for a new name
		name, err := callOpenAI(animalNames)
		if err != nil {
			logging.Error("hybrid-name openai failed", err, logging.Fields{constants.LogFieldKey: sfKey})
			return genRes{}, err
		}
		if name == "" {
			logging.Error("hybrid-name openai returned empty name", fmt.Errorf("empty"), logging.Fields{constants.LogFieldKey: sfKey})
			return genRes{}, fmt.Errorf("openai returned empty name")
		}

		logging.Info("hybrid-name openai success", logging.Fields{constants.LogFieldKey: sfKey, constants.LogFieldName: name})

		// Persist the generated name for future reuse. SaveGeneratedNameForAnimalIDs
		// will compute and store the canonical animal_key as well.
		if err := repo.SaveGeneratedNameForAnimalIDs(animalIDs, strings.Join(animalNames, " + "), name); err != nil {
			logging.Error("hybrid-name failed to save generated name", err, logging.Fields{constants.LogFieldKey: sfKey})
		} else {
			logging.Info("hybrid-name saved generated name", logging.Fields{constants.LogFieldKey: sfKey})
		}

		return genRes{Name: name, Source: "openai"}, nil
	})

	// Wait for the singleflight result, but don't wait indefinitely.
	select {
	case r := <-ch:
		if r.Err != nil {
			logging.Error("hybrid-name generation failed (singleflight)", r.Err, logging.Fields{constants.LogFieldKey: sfKey})
			return "", "openai_error", r.Err
		}
		if rr, ok := r.Val.(genRes); ok {
			return rr.Name, rr.Source, nil
		}
		// Unexpected type
		return "", "openai_error", fmt.Errorf("unexpected result type from singleflight")
	case <-time.After(60 * time.Second):
		logging.Error("hybrid-name generation timed out", fmt.Errorf("timeout"), logging.Fields{constants.LogFieldKey: sfKey})
		return "", "timeout", fmt.Errorf("timed out waiting for name generation")
	}
}
