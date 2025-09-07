package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/ericogr/chimera-cards/internal/game"
)

type entityEntry struct {
	Name             string           `json:"name"`
	HitPoints        int              `json:"hit_points"`
	Attack           int              `json:"attack"`
	Defense          int              `json:"defense"`
	Agility          int              `json:"agility"`
	Energy           int              `json:"energy"`
	VigorCost        int              `json:"vigor_cost"`
	SkillName        string           `json:"skill_name"`
	SkillCost        int              `json:"skill_cost"`
	SkillDescription string           `json:"skill_description"`
	SkillKey         string           `json:"skill_key"`
	SkillEffect      game.SkillEffect `json:"skill_effect"`
}

type rawConfig struct {
	EntityList []entityEntry `json:"entity_list"`
    Server     *struct {
        Address string `json:"address"`
    } `json:"server"`
	// Optional image prompt template used to generate entity/hybrid images.
	// Use the string token {{entities}} where the comma-separated list of
	// entity names will be substituted. If not provided, a sensible
	// default is used by the OpenAI client.
	// Optional single-entity image prompt template loaded from config
	SingleImagePrompt string `json:"single_image_prompt"`
	HybridImagePrompt string `json:"hybrid_image_prompt"`
    // Optional name prompt template used to generate hybrid names.
    // Use the token {{entities}} where the comma-separated list of entity
    // names will be substituted. If omitted, a default prompt is used.
    NamePrompt string `json:"name_prompt"`
    // Optional TTL controlling how long newly created public games remain
    // listed. Accepts a Go duration string (e.g. "5m", "30s") or an
    // integer number of seconds as fallback.
    PublicGamesTTL string `json:"public_games_ttl"`
}

// LoadedConfig contains entities to seed and the server address to bind to.
type LoadedConfig struct {
	Entities      []game.Entity
	ServerAddress string
	// Optional image prompt template loaded from config
	// Optional single-entity image prompt template loaded from config
	SingleImagePromptTemplate string
	// Optional hybrid image prompt template loaded from config
	HybridImagePromptTemplate string
	// Optional name prompt template loaded from config
	NamePromptTemplate string
	// How long to keep public games listed (duration)
	PublicGamesTTL time.Duration
}

// LoadConfig reads the configuration file at path and returns entities and
// server address. It requires the key `entity_list` (snake_case).
func LoadConfig(path string) (*LoadedConfig, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}
	var rc rawConfig
	if err := json.Unmarshal(b, &rc); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	entries := rc.EntityList
	if len(entries) == 0 {
		return nil, fmt.Errorf("config file %s: entity_list is empty (provide 'entity_list' array)", path)
	}
	out := make([]game.Entity, 0, len(entries))
	for _, a := range entries {
		if a.Name == "" {
			return nil, fmt.Errorf("config file %s: entity entry missing 'name'", path)
		}
		out = append(out, game.Entity{
			Name:             a.Name,
			HitPoints:        a.HitPoints,
			Attack:           a.Attack,
			Defense:          a.Defense,
			Agility:          a.Agility,
			Energy:           a.Energy,
			VigorCost:        a.VigorCost,
			SkillName:        a.SkillName,
			SkillCost:        a.SkillCost,
			SkillDescription: a.SkillDescription,
			SkillKey:         a.SkillKey,
			SkillEffect:      a.SkillEffect,
		})
	}

	// Cross-entry validation: ensure unique entity names (case-insensitive)
	// and unique skill_key values. Also enforce that any ability marked to
	// execute as a plan provides a non-empty skill_key so the engine can
	// route execution.
	nameSet := make(map[string]struct{}, len(out))
	skillSet := make(map[string]struct{}, len(out))
	for _, aa := range out {
		ln := strings.ToLower(strings.TrimSpace(aa.Name))
		if _, exists := nameSet[ln]; exists {
			return nil, fmt.Errorf("config file %s: duplicate entity name '%s'", path, aa.Name)
		}
		nameSet[ln] = struct{}{}
		if aa.SkillEffect.ExecutesPlan {
			if strings.TrimSpace(aa.SkillKey) == "" {
				return nil, fmt.Errorf("config file %s: entity '%s' marked executes_plan but missing 'skill_key'", path, aa.Name)
			}
		}
		if aa.SkillKey != "" {
			if _, exists := skillSet[aa.SkillKey]; exists {
				return nil, fmt.Errorf("config file %s: duplicate skill_key '%s'", path, aa.SkillKey)
			}
			skillSet[aa.SkillKey] = struct{}{}
		}
	}

	addr := ":8080"
	if rc.Server != nil && rc.Server.Address != "" {
		addr = rc.Server.Address
	}

    // Parse public games TTL: default to 5 minutes if omitted or invalid.
    defaultTTL := 5 * time.Minute
    ttl := defaultTTL
    if strings.TrimSpace(rc.PublicGamesTTL) != "" {
        txt := strings.TrimSpace(rc.PublicGamesTTL)
        if d, err := time.ParseDuration(txt); err == nil {
            ttl = d
        } else {
            // Allow numeric seconds as a fallback (e.g. 300)
            if s, serr := strconv.Atoi(txt); serr == nil {
                ttl = time.Duration(s) * time.Second
            } else {
                return nil, fmt.Errorf("config file %s: invalid public_games_ttl: %w", path, err)
            }
        }
    }

	return &LoadedConfig{
		Entities:                  out,
		ServerAddress:             addr,
		SingleImagePromptTemplate: strings.TrimSpace(rc.SingleImagePrompt),
		HybridImagePromptTemplate: strings.TrimSpace(rc.HybridImagePrompt),
		NamePromptTemplate:        strings.TrimSpace(rc.NamePrompt),
		PublicGamesTTL:            ttl,
	}, nil
}

// (No compatibility wrapper) Use LoadConfig to obtain entities and server address.
