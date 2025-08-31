package storage

import "github.com/ericogr/quimera-cards/internal/game"

type Repository interface {
	GetAnimals() ([]game.Animal, error)
	GetPublicGames() ([]game.Game, error)
	CreateGame(g *game.Game) error
	GetGameByID(id uint) (*game.Game, error)
	FindGameByJoinCode(code string) (*game.Game, error)
	UpdateGame(g *game.Game) error
	GetAnimalsByIDs(ids []uint) ([]game.Animal, error)
	// SaveAnimalImage stores a PNG blob for the given animal ID.
	SaveAnimalImage(animalID uint, pngBytes []byte) error

	// GetAnimalByName returns an animal by its name (case-insensitive).
	GetAnimalByName(name string) (*game.Animal, error)
	// Generated name cache (lookup by canonical animal key)
	// e.g. key = "lion_raven"
	GetGeneratedNameByAnimalKey(key string) (*game.HybridGeneratedName, error)
	SaveGeneratedNameForAnimalIDs(ids []uint, animalNames, generatedName string) error
	// Hybrid image storage
	GetHybridImageByKey(key string) ([]byte, error)
	SaveHybridImageByKey(key string, png []byte) error
	RemovePlayerByUUID(gameID uint, playerUUID string) error
	UpsertUser(email, uuid, name string) error
	UpdateStatsOnGameEnd(g *game.Game, resignedEmail string) error
	GetStatsByEmail(email string) (*game.User, error)
	SaveUser(u *game.User) error
	// Leaderboard
	GetTopPlayers(limit int) ([]game.User, error)
}
