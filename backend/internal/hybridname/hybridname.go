package hybridname

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ericogr/quimera-cards/internal/constants"
	"github.com/ericogr/quimera-cards/internal/dedupe"
	"github.com/ericogr/quimera-cards/internal/logging"

	"github.com/ericogr/quimera-cards/internal/keys"
	"github.com/ericogr/quimera-cards/internal/storage"
)

// namePromptTemplate can be set at application startup to customize the
// prompt used when requesting hybrid names from OpenAI. Use the token
// "{{entities}}" where the comma-separated list of entity names will be
// substituted.
var namePromptTemplate string

// SetNamePromptTemplate sets a custom prompt template for hybrid name
// generation. Call from main after loading configuration.
func SetNamePromptTemplate(t string) {
	namePromptTemplate = strings.TrimSpace(t)
}

// buildKeyFromIDs returns a canonical key for a list of entity IDs, e.g. "1,3,7".
// buildKeyFromIDs removed; we canonicalize by names via keys.EntityKeyFromNames

// callOpenAI invokes the OpenAI Chat Completions API to generate a single
// creative name for the provided entity names. It returns the generated name
// or an error if the request failed.
func callOpenAI(entityNames []string) (string, error) {
	apiKey := os.Getenv(constants.EnvOpenAIAPIKey)
	if apiKey == "" {
		return "", fmt.Errorf("%s not set", constants.EnvOpenAIAPIKey)
	}

	// Build prompt from template. If a configured template is present use
	// it; otherwise fall back to a sensible default. The template should
	// contain the token {{entities}} where the names list will be inserted.
    entitiesPart := strings.Join(entityNames, ", ")
    prompt := namePromptTemplate
	if prompt == "" {
		prompt = "Given these entity names: {{entities}}. Create a short, fun, single-name hybrid that combines them (1-3 words). Return only the name."
	}
    prompt = strings.ReplaceAll(prompt, "{{entities}}", entitiesPart)

    // Log the prompt so operators can see exactly what was sent to OpenAI
    logging.Info("hybrid-name openai prompt", logging.Fields{"entities": entitiesPart, "prompt": prompt})

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
// for the given entity IDs; if not found, it calls OpenAI to generate one and
// stores it in the repository. It returns the name, the source ("db"|"openai"),
// and an error if the OpenAI call failed.
func GetOrCreateGeneratedName(repo storage.Repository, entityNames []string) (string, string, error) {
    // Build canonical entity key from names: lowercase, underscores, sorted.
    entityKey := keys.EntityKeyFromNames(entityNames)

	// Try cache by canonical name-key first.
    if entityKey != "" {
        if gn, err := repo.GetGeneratedNameByEntityKey(entityKey); err == nil && gn != nil && gn.GeneratedName != "" {
            logging.Info("hybrid-name cache hit by entity_key", logging.Fields{constants.LogFieldKey: entityKey, constants.LogFieldName: gn.GeneratedName, constants.LogFieldSource: "db_key"})
            return gn.GeneratedName, "db_key", nil
        }
    }

	// Not cached — deduplicate concurrent generation using singleflight
    // keyed by the canonical entityKey (fallback to a stable string if
    // entityKey is empty).
    sfKey := entityKey
    if sfKey == "" {
        // As a last resort use the joined entity names string (unsorted)
        sfKey = strings.Join(entityNames, " + ")
    }

	type genRes struct {
		Name   string
		Source string
	}

	ch := dedupe.NameGroup.DoChan(sfKey, func() (interface{}, error) {
        // Re-check DB by entity key inside the singleflight function in
        // case another goroutine saved the generated name before we got here.
        if entityKey != "" {
            if gn, err := repo.GetGeneratedNameByEntityKey(entityKey); err == nil && gn != nil && gn.GeneratedName != "" {
                logging.Info("hybrid-name cache hit (singleflight)", logging.Fields{constants.LogFieldKey: entityKey, constants.LogFieldName: gn.GeneratedName, constants.LogFieldSource: "db_key"})
                return genRes{Name: gn.GeneratedName, Source: "db_key"}, nil
            }
        }

		// Ask OpenAI for a new name
		name, err := callOpenAI(entityNames)
		if err != nil {
			logging.Error("hybrid-name openai failed", err, logging.Fields{constants.LogFieldKey: sfKey})
			return genRes{}, err
		}
		if name == "" {
			logging.Error("hybrid-name openai returned empty name", fmt.Errorf("empty"), logging.Fields{constants.LogFieldKey: sfKey})
			return genRes{}, fmt.Errorf("openai returned empty name")
		}

		logging.Info("hybrid-name openai success", logging.Fields{constants.LogFieldKey: sfKey, constants.LogFieldName: name})

		// Persist the generated name for future reuse.
        // Attempt to resolve numeric IDs from names so we can save the
        // canonical row with entity key and numeric foreign keys. If
        // any name is missing in the entities table we skip saving by IDs
        // (the name is still usable via the entity_key lookup).
		ids := make([]uint, 0, len(entityNames))
		for _, n := range entityNames {
			if a, err := repo.GetEntityByName(n); err == nil && a != nil {
				ids = append(ids, a.ID)
			} else {
				ids = nil
				break
			}
		}
        if ids != nil && (len(ids) == 2 || len(ids) == 3) {
            if err := repo.SaveGeneratedNameForEntityIDs(ids, strings.Join(entityNames, " + "), name); err != nil {
                logging.Error("hybrid-name failed to save generated name", err, logging.Fields{constants.LogFieldKey: sfKey})
            } else {
                logging.Info("hybrid-name saved generated name", logging.Fields{constants.LogFieldKey: sfKey})
            }
        } else {
            // Best-effort: save using entity_key only via repository by
            // attempting a direct create (if repository supported it).
            // For now we skip numeric-key save so cached lookup will rely
            // on the stored entity_key created when possible.
            logging.Info("hybrid-name saved to cache skipped (missing numeric ids)", logging.Fields{constants.LogFieldKey: sfKey})
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
