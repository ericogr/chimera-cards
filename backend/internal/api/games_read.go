package api

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

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
	out, err := MarshalIntoSnakeTimestamps(users)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedFetchLeaderboard})
		return
	}
	c.JSON(http.StatusOK, out)
}

// GetGame returns a game by ID.
func (h *GameHandler) GetGame(c *gin.Context) {
	// The route param contains the game's join code (string). Look up the
	// game by its join code and then load the full game by ID so the
	// returned payload includes preloaded hybrids and entity data.
	code := normalizeJoinCode(c.Param("gameCode"))
	if code == "" || !joinCodeRegex.MatchString(code) {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidGameID})
		return
	}
	short, err := h.repo.FindGameByJoinCode(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{constants.JSONKeyError: constants.ErrGameNotFound})
		return
	}
	g, err := h.repo.GetGameByID(short.ID)
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

// UpdatePlayerProfile updates the authenticated player's display name.
func (h *GameHandler) UpdatePlayerProfile(c *gin.Context) {
	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidRequest})
		return
	}
	// Require authenticated email from context (no fallbacks).
	email := ""
	if v, ok := c.Get("userEmail"); ok {
		email, _ = v.(string)
	}
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrEmailRequired})
		return
	}
	// Validate display name using the same Unicode-aware pattern as the
	// frontend. Accept letters, marks, numbers, apostrophe, dot, hyphen
	// and spaces, length 4-40.
	var playerNameRegex = regexp.MustCompile(`^[\p{L}\p{M}\p{N}.'\- ]{4,40}$`)

	trimmed := strings.TrimSpace(body.Name)
	if !playerNameRegex.MatchString(trimmed) {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: "Invalid player name"})
		return
	}

	// Load or create user stats record
	ps, err := h.repo.GetStatsByEmail(email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedFetchStats})
		return
	}
	ps.PlayerName = trimmed
	// Persist
	if err := h.repo.SaveUser(ps); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedUpdateGame})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
