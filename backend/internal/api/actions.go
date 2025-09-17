package api

import (
	"net/http"

	"github.com/ericogr/chimera-cards/internal/constants"
	"github.com/ericogr/chimera-cards/internal/game"
	"github.com/ericogr/chimera-cards/internal/service"

	"github.com/gin-gonic/gin"
)

type ActionRequest struct {
    ActionType string `json:"action_type"`
    EntityID   uint   `json:"entity_id"`
}

// SubmitAction stores a player's chosen action for the current round.
func (h *GameHandler) SubmitAction(c *gin.Context) {
	// Path param contains join code. Resolve to internal ID.
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
	if g.Status != game.StatusInProgress {
		c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrGameNotInProgress})
		return
	}
	if g.Phase != game.PhasePlanning {
		c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrActionsLockedResolvingRound})
		return
	}
	var req ActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidRequest})
		return
	}

	// Derive the calling player's UUID from the authenticated session
	userEmail, _ := c.Get("userEmail")
	emailStr, _ := userEmail.(string)
	if emailStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{constants.JSONKeyError: constants.ErrAuthRequired})
		return
	}
    // Ensure the session user is a participant
    found := false
    for i := range g.Players {
        if g.Players[i].PlayerEmail == emailStr {
            found = true
            break
        }
    }
    if !found {
        c.JSON(http.StatusForbidden, gin.H{constants.JSONKeyError: constants.ErrPlayerNotInThisGame})
        return
    }

    // Delegate to service layer using session email as identity
    actionType := game.PendingActionType(req.ActionType)
    g2, resolved, err := service.SubmitAction(h.repo, g.ID, emailStr, actionType, req.EntityID, h.actionTimeout)
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
