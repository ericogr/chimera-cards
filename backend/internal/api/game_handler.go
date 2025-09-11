package api

import (
	"net/http"
	"time"

	"github.com/ericogr/chimera-cards/internal/storage"
	"github.com/gin-gonic/gin"
)

// GameHandler groups all game-related HTTP handlers.
type GameHandler struct {
	repo           storage.Repository
	actionTimeout  time.Duration
	publicGamesTTL time.Duration
}

// NewGameHandler creates a new GameHandler with the given repository and
// configured per-round action timeout and public games TTL.
func NewGameHandler(repo storage.Repository, actionTimeout, publicGamesTTL time.Duration) *GameHandler {
	return &GameHandler{repo: repo, actionTimeout: actionTimeout, publicGamesTTL: publicGamesTTL}
}

// GetConfig returns runtime configuration values consumed by the frontend.
// It exposes the public games TTL and the per-round action timeout as
// integer seconds to simplify client parsing.
func (h *GameHandler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"public_games_ttl_seconds": int(h.publicGamesTTL.Seconds()),
		"action_timeout_seconds":   int(h.actionTimeout.Seconds()),
	})
}
