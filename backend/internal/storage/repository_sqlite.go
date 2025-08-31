package storage

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ericogr/quimera-cards/internal/game"
	"gorm.io/gorm"
)

type sqliteRepository struct {
	db *gorm.DB
	// configByName maps lowercase animal name -> config definition (stats).
	configByName map[string]game.Animal
}

func NewSQLiteRepository(db *gorm.DB, configAnimals []game.Animal) Repository {
	m := make(map[string]game.Animal, len(configAnimals))
	for _, a := range configAnimals {
		m[strings.ToLower(a.Name)] = a
	}
	return &sqliteRepository{db: db, configByName: m}
}

func (r *sqliteRepository) GetAnimals() ([]game.Animal, error) {
	var animals []game.Animal
	// Exclude the internal "None" animal from selection lists.
	if err := r.db.Where("name != ?", string(game.None)).Find(&animals).Error; err != nil {
		return nil, err
	}
	// Override stats from config when available (config is source of truth)
	for i := range animals {
		if r.configByName != nil {
			if conf, ok := r.configByName[strings.ToLower(animals[i].Name)]; ok {
				animals[i].HitPoints = conf.HitPoints
				animals[i].Attack = conf.Attack
				animals[i].Defense = conf.Defense
				animals[i].Agility = conf.Agility
				animals[i].Energy = conf.Energy
				animals[i].VigorCost = conf.VigorCost
				animals[i].SkillName = conf.SkillName
				animals[i].SkillCost = conf.SkillCost
				animals[i].SkillDescription = conf.SkillDescription
			}
		}
	}
	return animals, nil
}

func (r *sqliteRepository) CreateGame(g *game.Game) error {
	return r.db.Create(g).Error
}

func (r *sqliteRepository) GetGameByID(id uint) (*game.Game, error) {
	var g game.Game
	err := r.db.Preload("Players.Hybrids.BaseAnimals").First(&g, id).Error
	return &g, err
}

func (r *sqliteRepository) UpdateGame(g *game.Game) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(g).Error
}

func (r *sqliteRepository) GetAnimalsByIDs(ids []uint) ([]game.Animal, error) {
	var animals []game.Animal
	err := r.db.Where("id IN ?", ids).Find(&animals).Error
	if err != nil {
		return animals, err
	}
	// Override stats from config
	for i := range animals {
		if r.configByName != nil {
			if conf, ok := r.configByName[strings.ToLower(animals[i].Name)]; ok {
				animals[i].HitPoints = conf.HitPoints
				animals[i].Attack = conf.Attack
				animals[i].Defense = conf.Defense
				animals[i].Agility = conf.Agility
				animals[i].Energy = conf.Energy
				animals[i].VigorCost = conf.VigorCost
				animals[i].SkillName = conf.SkillName
				animals[i].SkillCost = conf.SkillCost
				animals[i].SkillDescription = conf.SkillDescription
			}
		}
	}
	return animals, nil
}

func (r *sqliteRepository) GetPublicGames() ([]game.Game, error) {
	var games []game.Game
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	if err := r.db.Preload("Players").Where("private = ? AND created_at > ?", false, fiveMinutesAgo).Order("created_at desc").Find(&games).Error; err != nil {
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
		Preload("Hybrids.BaseAnimals").First(&p).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, h := range p.Hybrids {
		if err := tx.Model(&h).Association("BaseAnimals").Clear(); err != nil {
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
		ps.PlayerName = name
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
	u.PlayerName = name
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

// SaveAnimalImage stores PNG bytes for the specified animal ID in the DB.
func (r *sqliteRepository) SaveAnimalImage(animalID uint, pngBytes []byte) error {
	return r.db.Model(&game.Animal{}).Where("id = ?", animalID).Update("image_png", pngBytes).Error
}

func (r *sqliteRepository) GetAnimalByName(name string) (*game.Animal, error) {
	var a game.Animal
	if err := r.db.Where("lower(name) = ?", strings.ToLower(name)).First(&a).Error; err != nil {
		return nil, err
	}
	if r.configByName != nil {
		if conf, ok := r.configByName[strings.ToLower(a.Name)]; ok {
			a.HitPoints = conf.HitPoints
			a.Attack = conf.Attack
			a.Defense = conf.Defense
			a.Agility = conf.Agility
			a.Energy = conf.Energy
			a.VigorCost = conf.VigorCost
			a.SkillName = conf.SkillName
			a.SkillCost = conf.SkillCost
			a.SkillDescription = conf.SkillDescription
		}
	}
	return &a, nil
}

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

func (r *sqliteRepository) GetGeneratedNameByAnimalIDs(ids []uint) (*game.HybridGeneratedName, error) {
	// Normalize and sort IDs so the lookup is order-independent, then map to
	// the three animal key columns. The third key may be nil for two-animal
	// combinations.
	if len(ids) < 2 || len(ids) > 3 {
		return nil, gorm.ErrRecordNotFound
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
		// Use 0 to represent the absent third animal.
		a3 = 0
	}

	var h game.HybridGeneratedName
	if err := r.db.Where("animal1_key = ? AND animal2_key = ? AND animal3_key = ?", a1, a2, a3).First(&h).Error; err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *sqliteRepository) SaveGeneratedNameForAnimalIDs(ids []uint, animalNames, generatedName string) error {
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

	// Build canonical animal key from the provided human-readable string.
	// The caller passes `animalNames` typically as "Name1 + Name2".
	parts := strings.Split(animalNames, " + ")
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		q := strings.TrimSpace(p)
		if q != "" {
			cleaned = append(cleaned, strings.ToLower(strings.ReplaceAll(q, " ", "_")))
		}
	}
	sort.Strings(cleaned)
	animalKey := strings.Join(cleaned, "_")

	h := game.HybridGeneratedName{Animal1Key: a1, Animal2Key: a2, Animal3Key: a3, AnimalNames: animalNames, GeneratedName: generatedName, AnimalKey: animalKey}
	return r.db.Create(&h).Error
}

func (r *sqliteRepository) GetHybridImageByKey(key string) ([]byte, error) {
	var h game.HybridGeneratedName
	if err := r.db.Where("animal_key = ?", key).First(&h).Error; err != nil {
		return nil, err
	}
	return h.ImagePNG, nil
}

func (r *sqliteRepository) SaveHybridImageByKey(key string, png []byte) error {
	// Try to update existing record first
	res := r.db.Model(&game.HybridGeneratedName{}).Where("animal_key = ?", key).Update("image_png", png)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		return nil
	}
	// Otherwise create a minimal record
	h := game.HybridGeneratedName{AnimalKey: key, ImagePNG: png}
	return r.db.Create(&h).Error
}
