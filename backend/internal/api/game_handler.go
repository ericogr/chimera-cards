package api

import (
	"time"

	"github.com/ericogr/chimera-cards/internal/storage"
)

// GameHandler groups all game-related HTTP handlers.
type GameHandler struct {
	repo          storage.Repository
	actionTimeout time.Duration
}

// NewGameHandler creates a new GameHandler with the given repository and
// configured per-round action timeout.
func NewGameHandler(repo storage.Repository, actionTimeout time.Duration) *GameHandler {
	return &GameHandler{repo: repo, actionTimeout: actionTimeout}
}
