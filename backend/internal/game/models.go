package game

import (
	"sort"

	"gorm.io/gorm"
)

type Entity struct {
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
	// ImagePNG stores the 256x256 PNG bytes for this entity. It is
	// intentionally omitted from JSON responses (`json:"-"`) and stored
	// as a BLOB in the database column `image_png`.
	ImagePNG         []byte `json:"-" gorm:"column:image_png;type:blob"`
	// Skill contains the nested skill configuration (human-friendly and
	// machine-readable parts). It is not persisted in the DB (gorm:"-")
	// but will be exposed in API responses as a nested object.
	Skill Skill `json:"skill" gorm:"-"`
}

// TableName overrides the default GORM table name for Entity so the
// persisted table is `entity_templates` instead of the default `entities`.
func (Entity) TableName() string { return "entity_templates" }

// SkillEffect is a flexible description of what an entity's special ability
// does in-game. All fields are optional and will be applied when present.
type SkillEffect struct {
	// Opponent stat modifiers
	OpponentAttackDebuffPercent  int `json:"opponent_attack_debuff_percent"`
	OpponentAttackDebuffDuration int `json:"opponent_attack_debuff_duration"`

	OpponentAgilityDebuffPercent  int `json:"opponent_agility_debuff_percent"`
	OpponentAgilityDebuffDuration int `json:"opponent_agility_debuff_duration"`

	// Self buffs
	AttackBuffPercent  int `json:"attack_buff_percent"`
	AttackBuffDuration int `json:"attack_buff_duration"`

	// When true the ability causes the attack to ignore the opponent's
	// defense this round (or for the configured duration if provided).
	AttackIgnoresDefense         bool `json:"attack_ignores_defense"`
	AttackIgnoresDefenseDuration int  `json:"attack_ignores_defense_duration"`


	// Defense multiplier applied to self for the given duration.
	DefenseBuffMultiplier int `json:"defense_buff_multiplier"`
	DefenseBuffDuration   int `json:"defense_buff_duration"`
	// Prevent this hybrid from attacking for the configured duration.
	CannotAttack         bool `json:"cannot_attack"`
	CannotAttackDuration int  `json:"cannot_attack_duration"`

	// Instant effects
	RestoreEnergy int `json:"restore_energy"`

	// Swift pounce specific options
	SwiftAddAgilityDivisor    int `json:"swift_add_agility_divisor"`
	SwiftIgnoreDefensePercent int `json:"swift_ignore_defense_percent"`

    // Note: previously this struct included a few execution-specific
    // parameters (priority, reveal, charge/stun execution flags). Those
    // options have been removed to simplify the ability system — remaining
    // fields describe pure buffs/debuffs and instant effects only.
}

// Skill is a compact wrapper combining the human-readable metadata for an
// ability (name, description, cost, key) with the structured machine
// parameters contained in SkillEffect. Keep it non-persistent.
type Skill struct {
    Name        string      `json:"name" gorm:"-"`
    Description string      `json:"description" gorm:"-"`
    Cost        int         `json:"cost" gorm:"-"`
    Key         string      `json:"key" gorm:"-"`
    Effect      SkillEffect `json:"effect" gorm:"-"`
}

type Hybrid struct {
	gorm.Model
	PlayerID uint `json:"-"`
	// Name is a derived, non-persistent display field built from the
	// hybrid's base entities (e.g. "Lion + Raven"). It is intentionally
	// ignored by GORM so the database does not store redundant data.
	Name string `json:"name" gorm:"-"`
	// GeneratedName is the AI-created final name for the hybrid. It is
	// empty until the game is started (both players created hybrids and
	// the host starts the match). The frontend shows this during combat;
	// during creation the `Name` field continues to hold the simple
	// concatenation (e.g. "Lion + Raven").
	GeneratedName string `json:"generated_name"`
	// Use a descriptive join table name for the many-to-many relation.
	BaseEntities     []Entity `json:"base_entities" gorm:"many2many:hybrid_base_entities;"`
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
	// SelectedAbilityEntityID stores the ID of the entity whose special ability
	// is available for this hybrid (chosen among its 2–3 base entities).
	SelectedAbilityEntityID *uint `json:"selected_ability_entity_id"`
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
	PendingActionEntityID *uint             `json:"pending_action_entity_id"`
}

// Store per-game participants in a dedicated table for clarity
func (Player) TableName() string { return "game_players" }

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

// Unify global users table name as "player_profiles"
func (User) TableName() string { return "player_profiles" }

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

// HybridGeneratedName stores AI-generated names for a canonical entity combination
// (identified by a sorted, comma-separated list of entity IDs). This allows
// the server to cache names produced by the OpenAI API and avoid duplicate
// calls for the same entity set.
type HybridGeneratedName struct {
	gorm.Model
	// Store up to three entity IDs in separate columns so lookups can use
	// explicit constraints. The third key uses 0 to represent "no entity",
	// which makes uniqueness constraints simpler (no NULLs).
	Entity1Key uint `json:"entity1_key" gorm:"column:entity1_key;uniqueIndex:idx_hybrid_generated_cache_entities"`
	Entity2Key uint `json:"entity2_key" gorm:"column:entity2_key;uniqueIndex:idx_hybrid_generated_cache_entities"`
	Entity3Key uint `json:"entity3_key" gorm:"column:entity3_key;uniqueIndex:idx_hybrid_generated_cache_entities"`

	// Associations to enforce foreign key constraints to the entities table.
	Entity1 Entity `gorm:"foreignKey:Entity1Key;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	Entity2 Entity `gorm:"foreignKey:Entity2Key;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	// Keep association pointer for convenience; when Entity3Key is 0 there is
	// intentionally no associated row.
	Entity3 Entity `gorm:"foreignKey:Entity3Key;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`

	GeneratedName string `json:"generated_name"`
	// Canonical key for this entity combination (names sorted alphabetically,
	// lowercase, separated by underscore). Used to lookup/store the hybrid
	// image associated with this entity set.
	EntityKey string `json:"entity_key" gorm:"uniqueIndex"`

	// ImagePNG stores the 256x256 PNG bytes for the hybrid image.
	ImagePNG []byte `json:"-" gorm:"column:image_png;type:blob"`
}

// TableName overrides the default GORM table name for HybridGeneratedName
// so the persisted table is `hybrid_generated_cache` instead of the default
// `hybrid_generated_names`.
func (HybridGeneratedName) TableName() string { return "hybrid_generated_cache" }

// BeforeSave is a GORM hook that ensures entity key columns are stored in
// ascending order (smallest ID first). This guarantees a canonical
// representation for a set of entities so database uniqueness constraints
// can be applied regardless of the order provided by the caller.
func (h *HybridGeneratedName) BeforeSave(tx *gorm.DB) (err error) {
	// Collect non-zero IDs so that 0 (meaning "none") is not considered in
	// the sorting. This ensures two-entity combinations are stored as
	// (min, max, 0) and three-entity combinations as (min, mid, max).
	ids := make([]uint, 0, 3)
	if h.Entity1Key != 0 {
		ids = append(ids, h.Entity1Key)
	}
	if h.Entity2Key != 0 {
		ids = append(ids, h.Entity2Key)
	}
	if h.Entity3Key != 0 {
		ids = append(ids, h.Entity3Key)
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	// Assign back. Fill missing slots with 0 so the DB column is always set.
	h.Entity1Key = 0
	h.Entity2Key = 0
	h.Entity3Key = 0
	if len(ids) > 0 {
		h.Entity1Key = ids[0]
	}
	if len(ids) > 1 {
		h.Entity2Key = ids[1]
	}
	if len(ids) > 2 {
		h.Entity3Key = ids[2]
	}
	return nil
}
