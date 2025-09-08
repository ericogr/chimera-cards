package api

import (
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/ericogr/chimera-cards/internal/constants"
	"github.com/ericogr/chimera-cards/internal/game"
	"github.com/ericogr/chimera-cards/internal/logging"
	"github.com/ericogr/chimera-cards/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreateGamePayload struct {
	PlayerName  string `json:"player_name"`
	PlayerUUID  string `json:"player_uuid"`
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

	player1UUID := req.PlayerUUID
	if player1UUID == "" {
		player1UUID = uuid.New().String()
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
		Status:      "waiting_for_players",
		JoinCode:    joinCode,
		Players: []game.Player{
			{PlayerUUID: player1UUID, PlayerName: req.PlayerName, PlayerEmail: req.PlayerEmail},
		},
		Message: "Game created. Waiting for second player.",
	}

	// Upsert user profile (name/email)
	_ = h.repo.UpsertUser(req.PlayerEmail, player1UUID, req.PlayerName)

	if err := h.repo.CreateGame(&newGame); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedCreateGame})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"game_id":      newGame.ID,
		"join_code":    joinCode,
		"creator_uuid": player1UUID,
	})
}

type JoinGamePayload struct {
	JoinCode    string `json:"join_code"`
	PlayerName  string `json:"player_name"`
	PlayerUUID  string `json:"player_uuid"`
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

	g, err := h.repo.FindGameByJoinCode(req.JoinCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{constants.JSONKeyError: constants.ErrGameNotFound})
		return
	}

	if len(g.Players) >= 2 {
		c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrGameFull})
		return
	}

	newPlayerUUID := req.PlayerUUID
	if newPlayerUUID == "" {
		newPlayerUUID = uuid.New().String()
	}
	newPlayer := game.Player{PlayerUUID: newPlayerUUID, PlayerName: req.PlayerName, PlayerEmail: req.PlayerEmail}

	g.Players = append(g.Players, newPlayer)
	g.Status = "waiting_for_players"
	g.Message = "Second player joined. Waiting for the game to start."

	// Upsert user profile (name/email)
	_ = h.repo.UpsertUser(req.PlayerEmail, newPlayerUUID, req.PlayerName)

	if err := h.repo.UpdateGame(g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedUpdateGame})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"game_id":     g.ID,
		"player_uuid": newPlayerUUID,
		"message":     "Successfully joined game",
	})
}

// StartGame initializes state for the first round.
func (h *GameHandler) StartGame(c *gin.Context) {
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
	if g.Status != "waiting_for_players" {
		c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrGameAlreadyStartingOrStarted})
		return
	}

	// Persist the "starting" state so other clients polling the game see
	// that hybrid creation is in progress.
	g.Status = "starting"
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
			gg.Status = "error"
			gg.Message = "Failed to create hybrid names or images. Please try again."
			_ = h.repo.UpdateGame(gg)
			return
		}
	}(g.ID)

	c.JSON(http.StatusAccepted, gin.H{"message": "Game starting"})
}

type LeaveGamePayload struct {
	PlayerUUID string `json:"player_uuid"`
}

type EndGamePayload struct {
	PlayerUUID  string `json:"player_uuid"`
	PlayerEmail string `json:"player_email"`
}

// LeaveGame removes a player from a waiting room.
func (h *GameHandler) LeaveGame(c *gin.Context) {
	gameID, err := strconv.Atoi(c.Param("gameID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidGameID})
		return
	}
	var req LeaveGamePayload
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerUUID == "" {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidRequest})
		return
	}
	g, err := h.repo.GetGameByID(uint(gameID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{constants.JSONKeyError: constants.ErrGameNotFound})
		return
	}
	if g.Status != "waiting_for_players" {
		c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrCannotLeaveAfterGameStarted})
		return
	}
	// Ensure the player belongs to the game
	found := false
	for _, p := range g.Players {
		if p.PlayerUUID == req.PlayerUUID {
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrPlayerNotInThisGame})
		return
	}
	if err := h.repo.RemovePlayerByUUID(g.ID, req.PlayerUUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedRemovePlayer})
		return
	}
	// Reflect removal in the in-memory model to avoid re-attaching via FullSaveAssociations
	filtered := make([]game.Player, 0, len(g.Players))
	for _, p := range g.Players {
		if p.PlayerUUID != req.PlayerUUID {
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
	// Default resolution on end
	g.Status = "finished"
	g.Phase = "resolved"
	g.Winner = ""

	// If a player is specified, count it as a resignation for that player.
	// Do NOT assign victory to the opponent: resignations only increment
	// the quitter's resignation stat and do not award a win to anyone.
	var req EndGamePayload
	_ = c.ShouldBindJSON(&req) // optional body; ignore errors
	if req.PlayerUUID != "" && len(g.Players) == 2 {
		var loser *game.Player
		if g.Players[0].PlayerUUID == req.PlayerUUID {
			loser = &g.Players[0]
		} else if g.Players[1].PlayerUUID == req.PlayerUUID {
			loser = &g.Players[1]
		}
		if loser != nil {
			// Mark a human-friendly message that someone resigned. Do not set
			// `g.Winner` so the opponent does not receive a win.
			g.Message = "Player resigned: " + loser.PlayerName
		}
	}
	if g.Message == "" {
		g.Message = "Game ended by a player"
	}
	// Update stats on resignation if not already counted
	if !g.StatsCounted {
		resignedEmail := req.PlayerEmail
		if resignedEmail == "" && req.PlayerUUID != "" && len(g.Players) == 2 {
			if g.Players[0].PlayerUUID == req.PlayerUUID {
				resignedEmail = g.Players[0].PlayerEmail
			} else if g.Players[1].PlayerUUID == req.PlayerUUID {
				resignedEmail = g.Players[1].PlayerEmail
			}
		}
		_ = h.repo.UpdateStatsOnGameEnd(g, resignedEmail)
		g.StatsCounted = true
	}
	if err := h.repo.UpdateGame(g); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedEndGame})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Game ended"})
}
