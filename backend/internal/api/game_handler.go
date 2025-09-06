package api

import "github.com/ericogr/chimera-cards/internal/storage"

// GameHandler groups all game-related HTTP handlers.
type GameHandler struct {
	repo storage.Repository
}

// NewGameHandler creates a new GameHandler with the given repository.
func NewGameHandler(repo storage.Repository) *GameHandler {
	return &GameHandler{repo: repo}
}
