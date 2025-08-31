package game

import (
	"sort"

	"gorm.io/gorm"
)

type Animal struct {
	gorm.Model
	Name string `json:"name"`
	// The following fields are configured via the server config (chimera_config.json)
	// and should NOT be persisted in the database. Mark them with `gorm:"-"`
	// so GORM ignores them for schema/migration purposes while keeping the
	// fields available in-memory and in JSON responses.
	HitPoints int `json:"pv" gorm:"-"`
	Attack    int `json:"atq" gorm:"-"`
	Defense   int `json:"def" gorm:"-"`
	Agility   int `json:"agi" gorm:"-"`
	Energy    int `json:"ene" gorm:"-"`
	VigorCost int `json:"vigor_cost" gorm:"-"`
	// ImagePNG stores the 256x256 PNG bytes for this animal. It is
	// intentionally omitted from JSON responses (`json:"-"`) and stored
	// as a BLOB in the database column `image_png`.
	ImagePNG         []byte `json:"-" gorm:"column:image_png;type:blob"`
	SkillName        string `json:"skill_name" gorm:"-"`
	SkillCost        int    `json:"skill_cost" gorm:"-"`
	SkillDescription string `json:"skill_description" gorm:"-"`
}

// TableName overrides the default GORM table name for Animal so the
// persisted table is `animals_generated` instead of the default `animals`.
func (Animal) TableName() string { return "animals_generated" }

type Hybrid struct {
	gorm.Model
	PlayerID uint `json:"-"`
	// Name is a derived, non-persistent display field built from the
	// hybrid's base animals (e.g. "Lion + Raven"). It is intentionally
	// ignored by GORM so the database does not store redundant data.
	Name string `json:"name" gorm:"-"`
	// GeneratedName is the AI-created final name for the hybrid. It is
	// empty until the game is started (both players created hybrids and
	// the host starts the match). The frontend shows this during combat;
	// during creation the `Name` field continues to hold the simple
	// concatenation (e.g. "Lion + Raven").
	GeneratedName string `json:"generated_name"`
	// Use a descriptive join table name for the many-to-many relation.
	BaseAnimals      []Animal `json:"base_animals" gorm:"many2many:hybrid_animals_combination;"`
	BaseHitPoints    int      `json:"base_pv"`
	CurrentHitPoints int      `json:"current_pv"`
	BaseAttack       int      `json:"base_atq"`
	CurrentAttack    int      `json:"current_atq"`
	BaseDefense      int      `json:"base_def"`
	CurrentDefense   int      `json:"current_def"`
	BaseAgility      int      `json:"base_agi"`
	CurrentAgility   int      `json:"current_agi"`
	BaseEnergy       int      `json:"base_ene"`
	CurrentEnergy    int      `json:"current_ene"`
	BaseVIG          int      `json:"base_vig"`
	CurrentVIG       int      `json:"current_vig"`
	// SelectedAbilityAnimalID stores the ID of the animal whose special ability
	// is available for this hybrid (chosen among its 2–3 base animals).
	SelectedAbilityAnimalID *uint `json:"selected_ability_animal_id"`
	IsActive                bool  `json:"is_active"`
	IsDefeated              bool  `json:"is_defeated"`

	StunnedUntilRound             int    `json:"stunned_until_round"`
	AttackDebuffPercent           int    `json:"attack_debuff_percent"`
	AttackDebuffUntilRound        int    `json:"attack_debuff_until_round"`
	AttackBuffPercent             int    `json:"attack_buff_percent"`
	AttackBuffUntilRound          int    `json:"attack_buff_until_round"`
	DefenseBuffMultiplier         int    `json:"defense_buff_multiplier"`
	DefenseBuffUntilRound         int    `json:"defense_buff_until_round"`
	DefendStanceActive            bool   `json:"defend_stance_active"`
	PriorityNextRound             bool   `json:"priority_next_round"`
	AgilityDebuffPercent          int    `json:"agility_debuff_percent"`
	AgilityDebuffUntilRound       int    `json:"agility_debuff_until_round"`
	CannotAttackUntilRound        int    `json:"cannot_attack_until_round"`
	SelfDefenseIgnoredUntilRound  int    `json:"self_defense_ignored_until_round"`
	LastAction                    string `json:"last_action"`
	AttackHalvedThisRound         bool   `json:"attack_halved_this_round"`
	VulnerableThisRound           bool   `json:"vulnerable_this_round"`
	AttackIgnoresDefenseThisRound bool   `json:"attack_ignores_defense_this_round"`
}

type Player struct {
	gorm.Model
	GameID                uint              `json:"-"`
	PlayerUUID            string            `json:"player_uuid"`
	PlayerName            string            `json:"player_name"`
	PlayerEmail           string            `json:"player_email"`
	Hybrids               []Hybrid          `json:"hybrids"`
	HasCreated            bool              `json:"has_created"`
	HasSubmittedAction    bool              `json:"has_submitted_action"`
	PendingActionType     PendingActionType `json:"pending_action_type"`
	PendingActionAnimalID *uint             `json:"pending_action_animal_id"`
}

// Store per-game participants in a dedicated table for clarity
func (Player) TableName() string { return "game_participants" }

type Game struct {
	gorm.Model
	Name             string   `json:"name" gorm:"size:32"`
	Description      string   `json:"description" gorm:"size:256"`
	Private          bool     `json:"private"`
	JoinCode         string   `json:"join_code" gorm:"unique"`
	Players          []Player `json:"players"`
	CurrentTurn      string   `json:"current_turn"`
	RoundCount       int      `json:"round_count"`
	TurnNumber       int      `json:"turn_number"`
	Phase            string   `json:"phase"` // planning | resolving
	Status           string   `json:"status"`
	Winner           string   `json:"winner"`
	Message          string   `json:"message"`
	LastRoundSummary string   `json:"last_round_summary"`
	StatsCounted     bool     `json:"-"`
}

// User stores unique player identity and aggregate stats.
type User struct {
	gorm.Model
	PlayerUUID   string `gorm:"index"`
	PlayerName   string
	Email        string `gorm:"uniqueIndex"`
	GamesPlayed  int
	Wins         int
	Resignations int
}

// Unify global users table name as "players"
func (User) TableName() string { return "players" }

// PendingActionType is a string alias representing a player's chosen action.
// Using a dedicated type instead of plain string makes code safer and self-documenting.
type PendingActionType string

const (
	PendingActionNone        PendingActionType = ""
	PendingActionBasicAttack PendingActionType = "basic_attack"
	PendingActionDefend      PendingActionType = "defend"
	PendingActionAbility     PendingActionType = "ability"
	PendingActionRest        PendingActionType = "rest"
	PendingActionSkip        PendingActionType = "skip"
)

// HybridGeneratedName stores AI-generated names for a canonical animal combination
// (identified by a sorted, comma-separated list of animal IDs). This allows
// the server to cache names produced by the OpenAI API and avoid duplicate
// calls for the same animal set.
type HybridGeneratedName struct {
	gorm.Model
	// Store up to three animal IDs in separate columns so lookups can use
	// explicit constraints. The third key uses 0 to represent "no animal",
	// which makes uniqueness constraints simpler (no NULLs).
	Animal1Key uint `json:"animal1_key" gorm:"column:animal1_key;uniqueIndex:idx_hybrid_animals"`
	Animal2Key uint `json:"animal2_key" gorm:"column:animal2_key;uniqueIndex:idx_hybrid_animals"`
	Animal3Key uint `json:"animal3_key" gorm:"column:animal3_key;uniqueIndex:idx_hybrid_animals"`

	// Associations to enforce foreign key constraints to the animals table.
	Animal1 Animal `gorm:"foreignKey:Animal1Key;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Animal2 Animal `gorm:"foreignKey:Animal2Key;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	// Keep association pointer for convenience; when Animal3Key is 0 there is
	// intentionally no associated row.
	Animal3 Animal `gorm:"foreignKey:Animal3Key;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`

	GeneratedName string `json:"generated_name"`
	// Canonical key for this animal combination (names sorted alphabetically,
	// lowercase, separated by underscore). Used to lookup/store the hybrid
	// image associated with this animal set.
	AnimalKey string `json:"animal_key" gorm:"uniqueIndex"`

	// ImagePNG stores the 256x256 PNG bytes for the hybrid image.
	ImagePNG []byte `json:"-" gorm:"column:image_png;type:blob"`
}

// TableName overrides the default GORM table name for HybridGeneratedName
// so the persisted table is `hybrid_generated` instead of the default
// `hybrid_generated_names`.
func (HybridGeneratedName) TableName() string { return "hybrid_generated" }

// BeforeSave is a GORM hook that ensures animal key columns are stored in
// ascending order (smallest ID first). This guarantees a canonical
// representation for a set of animals so database uniqueness constraints
// can be applied regardless of the order provided by the caller.
func (h *HybridGeneratedName) BeforeSave(tx *gorm.DB) (err error) {
	// Collect non-zero IDs so that 0 (meaning "none") is not considered in
	// the sorting. This ensures two-animal combinations are stored as
	// (min, max, 0) and three-animal combinations as (min, mid, max).
	ids := make([]uint, 0, 3)
	if h.Animal1Key != 0 {
		ids = append(ids, h.Animal1Key)
	}
	if h.Animal2Key != 0 {
		ids = append(ids, h.Animal2Key)
	}
	if h.Animal3Key != 0 {
		ids = append(ids, h.Animal3Key)
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	// Assign back. Fill missing slots with 0 so the DB column is always set.
	h.Animal1Key = 0
	h.Animal2Key = 0
	h.Animal3Key = 0
	if len(ids) > 0 {
		h.Animal1Key = ids[0]
	}
	if len(ids) > 1 {
		h.Animal2Key = ids[1]
	}
	if len(ids) > 2 {
		h.Animal3Key = ids[2]
	}
	return nil
}
