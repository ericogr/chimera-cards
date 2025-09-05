package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ericogr/quimera-cards/internal/game"
)

type animalEntry struct {
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
	AnimalList []animalEntry `json:"animal_list"`
	Server     *struct {
		Address string `json:"address"`
	} `json:"server"`
	// Optional image prompt template used to generate animal/hybrid images.
	// Use the string token {{animals}} where the comma-separated list of
	// animal names will be substituted. If not provided, a sensible
	// default is used by the OpenAI client.
	ImagePrompt string `json:"image_prompt"`
	// Optional name prompt template used to generate hybrid names.
	// Use the token {{animals}} where the comma-separated list of animal
	// names will be substituted. If omitted, a default prompt is used.
	NamePrompt string `json:"name_prompt"`
}

// LoadedConfig contains animals to seed and the server address to bind to.
type LoadedConfig struct {
	Animals       []game.Animal
	ServerAddress string
	// Optional image prompt template loaded from config
	ImagePromptTemplate string
	// Optional name prompt template loaded from config
	NamePromptTemplate string
}

// LoadConfig reads the configuration file at path and returns animals and
// server address. It requires the key `animal_list` (snake_case).
func LoadConfig(path string) (*LoadedConfig, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}
	var rc rawConfig
	if err := json.Unmarshal(b, &rc); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	entries := rc.AnimalList
	if len(entries) == 0 {
		return nil, fmt.Errorf("config file %s: animal_list is empty (provide 'animal_list' array)", path)
	}

	out := make([]game.Animal, 0, len(entries))
	for _, a := range entries {
		if a.Name == "" {
			return nil, fmt.Errorf("config file %s: animal entry missing 'name'", path)
		}
		out = append(out, game.Animal{
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

	// Cross-entry validation: ensure unique animal names (case-insensitive)
	// and unique skill_key values. Also enforce that any ability marked to
	// execute as a plan provides a non-empty skill_key so the engine can
	// route execution.
	nameSet := make(map[string]struct{}, len(out))
	skillSet := make(map[string]struct{}, len(out))
	for _, aa := range out {
		ln := strings.ToLower(strings.TrimSpace(aa.Name))
		if _, exists := nameSet[ln]; exists {
			return nil, fmt.Errorf("config file %s: duplicate animal name '%s'", path, aa.Name)
		}
		nameSet[ln] = struct{}{}
		if aa.SkillEffect.ExecutesPlan {
			if strings.TrimSpace(aa.SkillKey) == "" {
				return nil, fmt.Errorf("config file %s: animal '%s' marked executes_plan but missing 'skill_key'", path, aa.Name)
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

	return &LoadedConfig{
		Animals:             out,
		ServerAddress:       addr,
		ImagePromptTemplate: strings.TrimSpace(rc.ImagePrompt),
		NamePromptTemplate:  strings.TrimSpace(rc.NamePrompt),
	}, nil
}

// (No compatibility wrapper) Use LoadConfig to obtain animals and server address.
