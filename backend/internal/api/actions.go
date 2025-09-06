package api

import (
	"net/http"
	"strconv"

	"github.com/ericogr/quimera-cards/internal/constants"
	"github.com/ericogr/quimera-cards/internal/game"
	"github.com/ericogr/quimera-cards/internal/service"

	"github.com/gin-gonic/gin"
)

type ActionRequest struct {
	PlayerUUID string `json:"player_uuid"`
	ActionType string `json:"action_type"`
	EntityID   uint   `json:"entity_id"`
}

// SubmitAction stores a player's chosen action for the current round.
func (h *GameHandler) SubmitAction(c *gin.Context) {
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
	if g.Status != "in_progress" {
		c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrGameNotInProgress})
		return
	}
	if g.Phase != "planning" {
		c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrActionsLockedResolvingRound})
		return
	}
	var req ActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidRequest})
		return
	}
	// Delegate to service layer
	// Convert raw action_type string to the typed PendingActionType
	actionType := game.PendingActionType(req.ActionType)
	g2, resolved, err := service.SubmitAction(h.repo, uint(gameID), req.PlayerUUID, actionType, req.EntityID)
	if err != nil {
		switch err {
		case service.ErrGameNotFound:
			c.JSON(http.StatusNotFound, gin.H{constants.JSONKeyError: constants.ErrGameNotFound})
			return
		case service.ErrGameNotInProgress:
			c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrGameNotInProgress})
			return
		case service.ErrActionsLocked:
			c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrActionsLockedResolvingRound})
			return
		case service.ErrPlayerNotInGame:
			c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrPlayerNotInGame})
			return
		case service.ErrNoActiveHybrid:
			c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrNoActiveHybrid})
			return
		case service.ErrHybridHasNoSelectedAbility, service.ErrAbilityMismatch:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedStoreAction})
			return
		}
	}

	if resolved {
		c.JSON(http.StatusOK, gin.H{"message": "Round resolved", "round": g2.RoundCount})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Action stored. Waiting for opponent."})
	}
}
