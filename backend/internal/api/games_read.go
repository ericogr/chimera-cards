package api

import (
	"net/http"
	"strconv"

	"github.com/ericogr/chimera-cards/internal/constants"
	"github.com/gin-gonic/gin"
)

// ListEntities returns all available entities.
func (h *GameHandler) ListEntities(c *gin.Context) {
	entities, err := h.repo.GetEntities()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedFetchEntities})
		return
	}
	c.JSON(http.StatusOK, entities)
}

// ListPublicGames returns all public games waiting for players or in progress.
func (h *GameHandler) ListPublicGames(c *gin.Context) {
	games, err := h.repo.GetPublicGames()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedFetchGames})
		return
	}
	out, err := MarshalIntoSnakeTimestamps(games)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedEncodeGames})
		return
	}
	c.JSON(http.StatusOK, out)
}

// ListLeaderboard returns the top players by wins (desc), limited to top 10 by default.
func (h *GameHandler) ListLeaderboard(c *gin.Context) {
	// optional ?limit=N
	limit := 10
	if s := c.Query("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	users, err := h.repo.GetTopPlayers(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedFetchLeaderboard})
		return
	}
	// Return as-is; frontend computes defeats = games_played - wins - resignations
	c.JSON(http.StatusOK, users)
}

// GetGame returns a game by ID.
func (h *GameHandler) GetGame(c *gin.Context) {
	gameID, err := strconv.Atoi(c.Param("gameID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidGameID})
		return
	}

	g, err := h.repo.GetGameByID(uint(gameID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{constants.JSONKeyError: constants.ErrGameNotFound})
		return
	}
	out, err := MarshalIntoSnakeTimestamps(g)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedEncodeGame})
		return
	}
	c.JSON(http.StatusOK, out)
}

// GetPlayerStats returns aggregated stats for a given player UUID.
func (h *GameHandler) GetPlayerStats(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		if v, ok := c.Get("userEmail"); ok {
			email, _ = v.(string)
		}
	}
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrEmailRequired})
		return
	}
	ps, err := h.repo.GetStatsByEmail(email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedFetchStats})
		return
	}
	c.JSON(http.StatusOK, ps)
}
