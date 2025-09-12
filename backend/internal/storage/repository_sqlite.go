package storage

import (
	"sort"
	"strings"
	"time"

	"github.com/ericogr/chimera-cards/internal/game"
	"github.com/ericogr/chimera-cards/internal/keys"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type sqliteRepository struct {
	db *gorm.DB
	// configByName maps lowercase entity name -> config definition (stats).
	configByName map[string]game.Entity

	// publicGamesTTL controls how long newly created public games remain
	// listed by GetPublicGames (e.g. 5m). This value is provided by the
	// server configuration (chimera_config.json).
	publicGamesTTL time.Duration
}

func NewSQLiteRepository(db *gorm.DB, configEntities []game.Entity, publicGamesTTL time.Duration) Repository {
	m := make(map[string]game.Entity, len(configEntities))
	for _, a := range configEntities {
		m[strings.ToLower(a.Name)] = a
	}
	return &sqliteRepository{db: db, configByName: m, publicGamesTTL: publicGamesTTL}
}

func (r *sqliteRepository) GetEntities() ([]game.Entity, error) {
	var entities []game.Entity
	// Exclude the internal "None" entity from selection lists.
	if err := r.db.Where("name != ?", string(game.None)).Find(&entities).Error; err != nil {
		return nil, err
	}
	// Override stats from config when available (config is source of truth)
	for i := range entities {
		if r.configByName != nil {
			if conf, ok := r.configByName[strings.ToLower(entities[i].Name)]; ok {
				entities[i].ApplyConfig(conf)
			}
		}
	}
	return entities, nil
}

func (r *sqliteRepository) CreateGame(g *game.Game) error {
	return r.db.Create(g).Error
}

func (r *sqliteRepository) GetGameByID(id uint) (*game.Game, error) {
	var g game.Game

	err := r.db.Preload("Players.Hybrids.BaseEntities").First(&g, id).Error
	if err != nil {
		return nil, err
	}
	// Override stats from config for preloaded base entities so the
	// frontend receives the complete entity information (skill name, costs, etc.).
	if r.configByName != nil {
		for pi := range g.Players {
			for hi := range g.Players[pi].Hybrids {
				for ai := range g.Players[pi].Hybrids[hi].BaseEntities {
					a := &g.Players[pi].Hybrids[hi].BaseEntities[ai]
					if conf, ok := r.configByName[strings.ToLower(a.Name)]; ok {
						a.ApplyConfig(conf)
					}
				}
			}
		}
	}

	// Compute the display name for hybrids on every load. The hybrid name
	// is a concatenation of its base entity names (sorted) but is not
	// persisted in the database anymore (it's derived). Populate the
	// `Name` field so API responses include it.
	for pi := range g.Players {
		for hi := range g.Players[pi].Hybrids {
			h := &g.Players[pi].Hybrids[hi]
			names := make([]string, len(h.BaseEntities))
			for ai := range h.BaseEntities {
				names[ai] = h.BaseEntities[ai].Name
			}
			sort.Slice(names, func(i, j int) bool { return strings.ToLower(names[i]) < strings.ToLower(names[j]) })
			h.Name = strings.Join(names, " + ")
		}
	}
	return &g, nil
}

func (r *sqliteRepository) UpdateGame(g *game.Game) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(g).Error
}

func (r *sqliteRepository) GetEntitiesByIDs(ids []uint) ([]game.Entity, error) {
	var entities []game.Entity
	err := r.db.Where("id IN ?", ids).Find(&entities).Error
	if err != nil {
		return entities, err
	}
	// Override stats from config
	for i := range entities {
		if r.configByName != nil {
			if conf, ok := r.configByName[strings.ToLower(entities[i].Name)]; ok {
				entities[i].ApplyConfig(conf)
			}
		}
	}
	return entities, nil
}

func (r *sqliteRepository) GetPublicGames() ([]game.Game, error) {
	var games []game.Game
	cutoff := time.Now().Add(-r.publicGamesTTL)
	if err := r.db.Preload("Players").Where("private = ? AND created_at > ?", false, cutoff).Order("created_at desc").Find(&games).Error; err != nil {
		return nil, err
	}
	// Only return games with at least one player
	filtered := make([]game.Game, 0, len(games))
	for i := range games {
		if len(games[i].Players) >= 1 {
			filtered = append(filtered, games[i])
		}
	}
	return filtered, nil
}

func (r *sqliteRepository) FindTimedOutGames(now time.Time) ([]game.Game, error) {
	var games []game.Game
	// Find games that are in progress, in planning phase and whose
	// action_deadline has passed.
	if err := r.db.Preload("Players.Hybrids.BaseEntities").Where("status = ? AND phase = ? AND action_deadline IS NOT NULL AND action_deadline <= ?", game.StatusInProgress, game.PhasePlanning, now).Find(&games).Error; err != nil {
		return nil, err
	}
	return games, nil
}

func (r *sqliteRepository) FindTimedOutGameIDs(now time.Time) ([]uint, error) {
	var ids []uint
	if err := r.db.Model(&game.Game{}).
		Where("status = ? AND phase = ? AND action_deadline IS NOT NULL AND action_deadline <= ?", game.StatusInProgress, game.PhasePlanning, now).
		Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *sqliteRepository) ClaimTimedOutGameIDs(now time.Time, limit int, reclaimAfter time.Duration, workerID string) ([]uint, error) {
	// processingAt marks when this worker claimed the rows
	processingAt := now
	reclaimThreshold := now.Add(-reclaimAfter)

	// Atomic-ish claim: update rows matching timeout conditions (and not
	// currently being processed, or whose processing_at is stale) and set
	// processing_by/processing_at to our worker id/time. Use a subselect
	// with ORDER BY action_deadline to claim the oldest first.
	sql := `UPDATE games SET processing_by = ?, processing_at = ? WHERE id IN (
        SELECT id FROM games
        WHERE status = ? AND phase = ? AND action_deadline IS NOT NULL AND action_deadline <= ?
          AND (processing_by IS NULL OR processing_at <= ?)
        ORDER BY action_deadline ASC
        LIMIT ?
    );`

	if err := r.db.Exec(sql, workerID, processingAt, game.StatusInProgress, game.PhasePlanning, now, reclaimThreshold, limit).Error; err != nil {
		return nil, err
	}

	// Now select the ids we claimed
	var ids []uint
	if err := r.db.Model(&game.Game{}).Where("processing_by = ? AND processing_at = ?", workerID, processingAt).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// Note: GetAllGames removed as unused to keep the repository lean.

func (r *sqliteRepository) FindGameByJoinCode(code string) (*game.Game, error) {
	var g game.Game
	err := r.db.Preload("Players").Where("join_code = ?", code).First(&g).Error
	return &g, err
}

func (r *sqliteRepository) RemovePlayerByUUID(gameID uint, playerUUID string) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	var p game.Player
	if err := tx.Where("game_id = ? AND player_uuid = ?", gameID, playerUUID).
		Preload("Hybrids.BaseEntities").First(&p).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, h := range p.Hybrids {
		if err := tx.Model(&h).Association("BaseEntities").Clear(); err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Where("player_id = ?", p.ID).Delete(&game.Hybrid{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Delete(&p).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *sqliteRepository) UpdateStatsOnGameEnd(g *game.Game, resignedEmail string) error {
	// Helper to upsert and add deltas
	upsert := func(email, uuid, name string, played, wins, resigns int) error {
		var ps game.User
		if err := r.db.Where("email = ?", email).First(&ps).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				ps = game.User{Email: email, PlayerUUID: uuid, PlayerName: name, GamesPlayed: 0, Wins: 0, Resignations: 0}
			} else {
				return err
			}
		}
		// Preserve an existing user-customized PlayerName. Only set the
		// PlayerName when creating a new record or when the stored name is empty.
		if ps.PlayerName == "" {
			ps.PlayerName = name
		}
		ps.PlayerUUID = uuid
		ps.GamesPlayed += played
		ps.Wins += wins
		ps.Resignations += resigns
		return r.db.Save(&ps).Error
	}
	if len(g.Players) != 2 {
		return nil
	}
	p1 := g.Players[0]
	p2 := g.Players[1]
	// everyone played one game
	if err := upsert(p1.PlayerEmail, p1.PlayerUUID, p1.PlayerName, 1, 0, 0); err != nil {
		return err
	}
	if err := upsert(p2.PlayerEmail, p2.PlayerUUID, p2.PlayerName, 1, 0, 0); err != nil {
		return err
	}
	// winner
	if g.Winner != "" {
		if p1.PlayerName == g.Winner {
			if err := upsert(p1.PlayerEmail, p1.PlayerUUID, p1.PlayerName, 0, 1, 0); err != nil {
				return err
			}
		} else if p2.PlayerName == g.Winner {
			if err := upsert(p2.PlayerEmail, p2.PlayerUUID, p2.PlayerName, 0, 1, 0); err != nil {
				return err
			}
		}
	}
	// resignation
	if resignedEmail != "" {
		if p1.PlayerEmail == resignedEmail {
			return upsert(p1.PlayerEmail, p1.PlayerUUID, p1.PlayerName, 0, 0, 1)
		}
		if p2.PlayerEmail == resignedEmail {
			return upsert(p2.PlayerEmail, p2.PlayerUUID, p2.PlayerName, 0, 0, 1)
		}
	}
	return nil
}

func (r *sqliteRepository) GetStatsByEmail(email string) (*game.User, error) {
	var ps game.User
	if err := r.db.Where("email = ?", email).First(&ps).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &game.User{Email: email, GamesPlayed: 0, Wins: 0, Resignations: 0}, nil
		}
		return nil, err
	}
	return &ps, nil
}

func (r *sqliteRepository) SaveUser(u *game.User) error {
	return r.db.Save(u).Error
}

func (r *sqliteRepository) UpsertUser(email, uuid, name string) error {
	var u game.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			u = game.User{Email: email, PlayerUUID: uuid, PlayerName: name}
		} else {
			return err
		}
	}
	// For existing users, preserve any user-customized PlayerName.
	// Only set the PlayerName when the stored value is empty.
	if u.PlayerName == "" {
		u.PlayerName = name
	}
	u.PlayerUUID = uuid
	return r.db.Save(&u).Error
}

// GetTopPlayers returns top N players ordered by Wins desc, then GamesPlayed desc
func (r *sqliteRepository) GetTopPlayers(limit int) ([]game.User, error) {
	if limit <= 0 {
		limit = 10
	}
	var users []game.User
	if err := r.db.Model(&game.User{}).
		Order("wins DESC").
		Order("games_played DESC").
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// SaveEntityImage stores PNG bytes for the specified entity ID in the DB.
func (r *sqliteRepository) SaveEntityImage(entityID uint, pngBytes []byte) error {
	return r.db.Model(&game.Entity{}).Where("id = ?", entityID).Update("image_png", pngBytes).Error
}

func (r *sqliteRepository) GetEntityByName(name string) (*game.Entity, error) {
	var a game.Entity
	if err := r.db.Where("lower(name) = ?", strings.ToLower(name)).First(&a).Error; err != nil {
		return nil, err
	}
	if r.configByName != nil {
		if conf, ok := r.configByName[strings.ToLower(a.Name)]; ok {
			a.ApplyConfig(conf)
		}
	}
	return &a, nil
}

// NOTE: lookup by numeric entity IDs was removed in favor of canonical
// name-key lookup (`GetGeneratedNameByEntityKey`). This keeps the cache
// stable across DB recreations where numeric IDs can change.

func (r *sqliteRepository) SaveGeneratedNameForEntityIDs(ids []uint, entityNames, generatedName string) error {
	if len(ids) < 2 || len(ids) > 3 {
		return gorm.ErrInvalidData
	}
	ints := make([]int, len(ids))
	for i, v := range ids {
		ints[i] = int(v)
	}
	sort.Ints(ints)

	a1 := uint(ints[0])
	a2 := uint(ints[1])
	var a3 uint
	if len(ints) == 3 {
		a3 = uint(ints[2])
	} else {
		a3 = 0
	}

	// Build canonical entity key from the provided human-readable string.
	// The caller passes `entityNames` typically as "Name1 + Name2".
	entityKey := keys.EntityKeyFromNames(strings.Split(entityNames, " + "))

	h := game.HybridGeneratedName{Entity1Key: a1, Entity2Key: a2, Entity3Key: a3, GeneratedName: generatedName, EntityKey: entityKey}
	// Use upsert semantics keyed by `entity_key` so that if a minimal record
	// was previously created (for example when saving an image only) we
	// update it with the generated name and numeric entity keys instead
	// of failing due to the unique constraint on `entity_key`.
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "entity_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"entity1_key", "entity2_key", "entity3_key", "generated_name"}),
	}).Create(&h).Error
}

func (r *sqliteRepository) GetHybridImageByKey(key string) ([]byte, error) {
	var h game.HybridGeneratedName
	if err := r.db.Where("entity_key = ?", key).First(&h).Error; err != nil {
		return nil, err
	}
	return h.ImagePNG, nil
}

func (r *sqliteRepository) SaveHybridImageByKey(key string, png []byte) error {
	// Try to update existing record first
	res := r.db.Model(&game.HybridGeneratedName{}).Where("entity_key = ?", key).Update("image_png", png)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		return nil
	}
	// Otherwise create a minimal record
	h := game.HybridGeneratedName{EntityKey: key, ImagePNG: png}
	return r.db.Create(&h).Error
}

// GetGeneratedNameByEntityKey looks up the generated hybrid name by the
// canonical entity key (lowercase names joined by underscores). This is a
// fallback when numeric IDs do not match cached rows (for example, when
// the database was recreated and entity IDs changed but names stayed the same).
func (r *sqliteRepository) GetGeneratedNameByEntityKey(key string) (*game.HybridGeneratedName, error) {
	var h game.HybridGeneratedName
	if err := r.db.Where("entity_key = ?", key).First(&h).Error; err != nil {
		return nil, err
	}
	return &h, nil
}
