package api

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ericogr/chimera-cards/internal/constants"
	"github.com/ericogr/chimera-cards/internal/engine"
	"github.com/ericogr/chimera-cards/internal/game"
	"github.com/ericogr/chimera-cards/internal/logging"
	"github.com/ericogr/chimera-cards/internal/service"
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
	out, err := MarshalForContext(c, games)
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
	out, err := MarshalForContext(c, users)
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
	// Log basic game state for debugging timeouts
	if len(g.Players) >= 2 {
		logging.Info("GET /api/games state", logging.Fields{
			constants.LogFieldGameID: g.ID,
			"phase":                  g.Phase,
			"action_deadline":        g.ActionDeadline,
			"p1_submitted":           g.Players[0].HasSubmittedAction,
			"p2_submitted":           g.Players[1].HasSubmittedAction,
		})
	} else {
		logging.Info("GET /api/games state (incomplete players)", logging.Fields{constants.LogFieldGameID: g.ID, "phase": g.Phase, "action_deadline": g.ActionDeadline})
	}

	// If the planning deadline has passed, attempt an immediate best-effort
	// resolution: if exactly one player is missing, auto-submit `rest` for
	// them so the round resolves immediately. If both players missed, end
	// the match as before. This helps clients that refresh the page right
	// after the deadline and avoids waiting for the background scanner.
	if g.Phase == game.PhasePlanning && !g.ActionDeadline.IsZero() && g.ActionDeadline.Before(time.Now()) {
		if len(g.Players) >= 2 {
			p1Submitted := g.Players[0].HasSubmittedAction
			p2Submitted := g.Players[1].HasSubmittedAction
			switch {
			case !p1Submitted && !p2Submitted:
				// both missed -> finish match immediately
				g.Status = game.StatusFinished
				g.Phase = game.PhaseResolved
				g.Winner = ""
				g.Message = "Match ended due to inactivity"
				g.LastRoundSummary = "Round timed out: both players failed to submit actions within the allotted time."
				g.StatsCounted = true
				g.ActionDeadline = time.Time{}
				if err := h.repo.UpdateGame(g); err != nil {
					logging.Error("failed to expire game (GET handler)", err, logging.Fields{constants.LogFieldGameID: g.ID})
				}
			case p1Submitted && !p2Submitted:
				// auto-submit REST for player 2
				if gr, ok := h.repo.(service.GameRepo); ok {
					_, _, serr := service.SubmitAction(gr, g.ID, g.Players[1].PlayerEmail, game.PendingActionRest, 0, h.actionTimeout)
					if serr != nil {
						logging.Error("GET handler failed to auto-submit rest", serr, logging.Fields{constants.LogFieldGameID: g.ID})
					}
					// reload
					if gg, err := h.repo.GetGameByID(short.ID); err == nil {
						g = gg
					}
				} else {
					// fallback inline
					g.Players[1].HasSubmittedAction = true
					g.Players[1].PendingActionType = game.PendingActionRest
					g.Players[1].PendingActionEntityID = nil
					engine.ResolveRound(g)
					if g.Status == game.StatusFinished {
						if !g.StatsCounted {
							_ = h.repo.UpdateStatsOnGameEnd(g, "")
							g.StatsCounted = true
						}
					} else {
						g.ActionDeadline = time.Now().Add(h.actionTimeout)
					}
					_ = h.repo.UpdateGame(g)
				}
			case !p1Submitted && p2Submitted:
				// auto-submit REST for player 1 (symmetric)
				if gr, ok := h.repo.(service.GameRepo); ok {
					_, _, serr := service.SubmitAction(gr, g.ID, g.Players[0].PlayerEmail, game.PendingActionRest, 0, h.actionTimeout)
					if serr != nil {
						logging.Error("GET handler failed to auto-submit rest", serr, logging.Fields{constants.LogFieldGameID: g.ID})
					}
					if gg, err := h.repo.GetGameByID(short.ID); err == nil {
						g = gg
					}
				} else {
					g.Players[0].HasSubmittedAction = true
					g.Players[0].PendingActionType = game.PendingActionRest
					g.Players[0].PendingActionEntityID = nil
					engine.ResolveRound(g)
					if g.Status == game.StatusFinished {
						if !g.StatsCounted {
							_ = h.repo.UpdateStatsOnGameEnd(g, "")
							g.StatsCounted = true
						}
					} else {
						g.ActionDeadline = time.Now().Add(h.actionTimeout)
					}
					_ = h.repo.UpdateGame(g)
				}
			}
		}
	}
	out, err := MarshalForContext(c, g)
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
	out, err := MarshalForContext(c, ps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedFetchStats})
		return
	}
	c.JSON(http.StatusOK, out)
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
