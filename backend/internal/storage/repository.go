package storage

import (
    "time"

    "github.com/ericogr/chimera-cards/internal/game"
)

type Repository interface {
	GetEntities() ([]game.Entity, error)
	GetPublicGames() ([]game.Game, error)
	CreateGame(g *game.Game) error
	GetGameByID(id uint) (*game.Game, error)
	FindGameByJoinCode(code string) (*game.Game, error)
	UpdateGame(g *game.Game) error
	GetEntitiesByIDs(ids []uint) ([]game.Entity, error)
	// SaveEntityImage stores a PNG blob for the given entity ID.
	SaveEntityImage(entityID uint, pngBytes []byte) error

	// GetEntityByName returns an entity by its name (case-insensitive).
	GetEntityByName(name string) (*game.Entity, error)
	// Generated name cache (lookup by canonical entity key)
	// e.g. key = "lion_raven"
	GetGeneratedNameByEntityKey(key string) (*game.HybridGeneratedName, error)
	SaveGeneratedNameForEntityIDs(ids []uint, entityNames, generatedName string) error
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
	// FindTimedOutGames returns games that are currently in-progress,
	// in the planning phase and whose action deadline is at or before
	// the provided time. The caller may then decide how to resolve them
	// (for example, marking them finished due to inactivity).
	FindTimedOutGames(now time.Time) ([]game.Game, error)
}
