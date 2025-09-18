package api

import (
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/ericogr/chimera-cards/internal/constants"
	"github.com/ericogr/chimera-cards/internal/game"
	"github.com/ericogr/chimera-cards/internal/logging"
	"github.com/ericogr/chimera-cards/internal/service"

	"github.com/gin-gonic/gin"
)

type CreateGamePayload struct {
	PlayerName  string `json:"player_name"`
	PlayerEmail string `json:"player_email"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
}

// CreateGame creates a new game and returns IDs and join code.
func (h *GameHandler) CreateGame(c *gin.Context) {
	var req CreateGamePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidRequest})
		return
	}
	// Derive identity from session
	if v, ok := c.Get("userEmail"); ok {
		req.PlayerEmail, _ = v.(string)
	}
	if v, ok := c.Get("userName"); ok && req.PlayerName == "" {
		req.PlayerName, _ = v.(string)
	}

	joinCode := generateJoinCode()

	// Validate lengths
	if utf8.RuneCountInString(req.Name) > 32 {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrGameNameExceeds})
		return
	}
	if utf8.RuneCountInString(req.Description) > 256 {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrDescriptionExceeds})
		return
	}

	newGame := game.Game{
		Name:        req.Name,
		Description: req.Description,
		Private:     req.Private,
		Status:      game.StatusWaitingForPlayers,
		JoinCode:    joinCode,
		Players: []game.Player{
			{PlayerName: req.PlayerName, PlayerEmail: req.PlayerEmail},
		},
		Message: "Game created. Waiting for second player.",
	}

	// Upsert user profile (name/email)
	_ = h.repo.UpsertUser(req.PlayerEmail, req.PlayerName)

	if err := h.repo.CreateGame(&newGame); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedCreateGame})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"game_id":   newGame.ID,
		"join_code": joinCode,
	})
}

type JoinGamePayload struct {
	JoinCode    string `json:"join_code"`
	PlayerName  string `json:"player_name"`
	PlayerEmail string `json:"player_email"`
}

// JoinGame allows a second player to join a game via join code.
func (h *GameHandler) JoinGame(c *gin.Context) {
	var req JoinGamePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidRequest})
		return
	}
	if v, ok := c.Get("userEmail"); ok {
		req.PlayerEmail, _ = v.(string)
	}
	if v, ok := c.Get("userName"); ok && req.PlayerName == "" {
		req.PlayerName, _ = v.(string)
	}

	code := normalizeJoinCode(req.JoinCode)
	if code == "" || !joinCodeRegex.MatchString(code) {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidGameID})
		return
	}
	g, err := h.repo.FindGameByJoinCode(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{constants.JSONKeyError: constants.ErrGameNotFound})
		return
	}

	if len(g.Players) >= 2 {
		c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrGameFull})
		return
	}

	newPlayer := game.Player{PlayerName: req.PlayerName, PlayerEmail: req.PlayerEmail}

	g.Players = append(g.Players, newPlayer)
	g.Status = game.StatusWaitingForPlayers
	g.Message = "Second player joined. Waiting for the game to start."

	// Upsert user profile (name/email)
	_ = h.repo.UpsertUser(req.PlayerEmail, req.PlayerName)

	if err := h.repo.UpdateGame(g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedUpdateGame})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"game_id":   g.ID,
		"join_code": g.JoinCode,
		"message":   "Successfully joined game",
	})
}

// StartGame initializes state for the first round.
func (h *GameHandler) StartGame(c *gin.Context) {
	// Resolve join code to internal ID
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

	if len(g.Players) < 2 {
		c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrNotEnoughPlayers})
		return
	}

	// Ensure both players created hybrids
	if len(g.Players) != 2 || !g.Players[0].HasCreated || !g.Players[1].HasCreated {
		c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrBothPlayersMustCreateHybrids})
		return
	}

	// Prevent starting if already in another transitional state
	if g.Status != game.StatusWaitingForPlayers {
		c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrGameAlreadyStartingOrStarted})
		return
	}

	// Persist the "starting" state so other clients polling the game see
	// that hybrid creation is in progress.
	g.Status = game.StatusStarting
	g.Message = "Your hybrid is being created. This may take a few moments."
	if err := h.repo.UpdateGame(g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedUpdateGameStatus})
		return
	}

	// Run the heavy work asynchronously so the request returns immediately
	// and both players can see the "starting" message while generation runs.
	go func(gameID uint) {
		// Reload full game state inside goroutine to avoid sharing the
		// handler's memory with the background worker.
		gg, err := h.repo.GetGameByID(gameID)
		if err != nil {
			logging.Error("async-start failed to load game", err, logging.Fields{constants.LogFieldGameID: gameID})
			return
		}
		if err := service.StartGame(h.repo, gg); err != nil {
			logging.Error("async-start failed to start game", err, logging.Fields{constants.LogFieldGameID: gameID})
			// Update game to a visible error state so players aren't left
			// waiting forever.
			gg.Status = game.StatusError
			gg.Message = "Failed to create hybrid names or images. Please try again."
			_ = h.repo.UpdateGame(gg)
			return
		}
		// Set initial action deadline for the first planning phase.
		gg.ActionDeadline = time.Now().Add(h.actionTimeout)
		_ = h.repo.UpdateGame(gg)
	}(g.ID)

	c.JSON(http.StatusAccepted, gin.H{"message": "Game starting"})
}

type LeaveGamePayload struct {
	// body intentionally empty; caller identity is derived from session
}

type EndGamePayload struct {
	PlayerEmail string `json:"player_email"`
}

// LeaveGame removes a player from a waiting room.
func (h *GameHandler) LeaveGame(c *gin.Context) {
	// find by join code
	code := normalizeJoinCode(c.Param("gameCode"))
	if code == "" || !joinCodeRegex.MatchString(code) {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidGameID})
		return
	}
	var req LeaveGamePayload
	// Body is optional; derive leaving player from authenticated session
	_ = c.ShouldBindJSON(&req)
	g, err := h.repo.FindGameByJoinCode(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{constants.JSONKeyError: constants.ErrGameNotFound})
		return
	}
	if g.Status != game.StatusWaitingForPlayers {
		c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrCannotLeaveAfterGameStarted})
		return
	}
	// Derive leaving player's UUID from session email and ensure they belong
	userEmail, _ := c.Get("userEmail")
	emailStr, _ := userEmail.(string)
	if emailStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{constants.JSONKeyError: constants.ErrAuthRequired})
		return
	}
	// Remove player by their email (derived from session)
	if err := h.repo.RemovePlayerByEmail(g.ID, emailStr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedRemovePlayer})
		return
	}
	// Reflect removal in the in-memory model to avoid re-attaching via FullSaveAssociations
	filtered := make([]game.Player, 0, len(g.Players))
	for _, p := range g.Players {
		if (p.PlayerEmail) != emailStr {
			filtered = append(filtered, p)
		}
	}
	g.Players = filtered
	// Optional: set message and keep game open for others
	g.Message = "A player left. Waiting for a new participant."
	if err := h.repo.UpdateGame(g); err != nil {
		// Not fatal for removing, but return error to client
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrPlayerRemovedFailedUpdate})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Player removed"})
}

// EndGame allows any player to end the match (cancels/finishes for both)
func (h *GameHandler) EndGame(c *gin.Context) {
	code := normalizeJoinCode(c.Param("gameCode"))
	if code == "" || !joinCodeRegex.MatchString(code) {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidGameID})
		return
	}
	g, err := h.repo.FindGameByJoinCode(code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{constants.JSONKeyError: constants.ErrGameNotFound})
		return
	}
	// Default resolution on end
	g.Status = game.StatusFinished
	g.Phase = game.PhaseResolved
	g.Winner = ""

	// If a player is specified, count it as a resignation for that player.
	// Do NOT assign victory to the opponent: resignations only increment
	// the quitter's resignation stat and do not award a win to anyone.
	var req EndGamePayload
	_ = c.ShouldBindJSON(&req) // optional body; ignore errors

	// Determine the caller via authenticated session. Only a participant
	// may resign the match on their behalf.
	userEmail, _ := c.Get("userEmail")
	emailStr, _ := userEmail.(string)
	if emailStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{constants.JSONKeyError: constants.ErrAuthRequired})
		return
	}

	var loser *game.Player
	for i := range g.Players {
		if g.Players[i].PlayerEmail == emailStr {
			loser = &g.Players[i]
			break
		}
	}
	if loser != nil {
		g.Message = "Player resigned: " + loser.PlayerName
	} else {
		// If caller is not a participant, forbid ending the match.
		c.JSON(http.StatusForbidden, gin.H{constants.JSONKeyError: constants.ErrPlayerNotInThisGame})
		return
	}

	if g.Message == "" {
		g.Message = "Game ended by a player"
	}

	// Update stats on resignation if not already counted
	if !g.StatsCounted {
		resignedEmail := loser.PlayerEmail
		_ = h.repo.UpdateStatsOnGameEnd(g, resignedEmail)
		g.StatsCounted = true
	}
	if err := h.repo.UpdateGame(g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedEndGame})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Game ended"})
}
