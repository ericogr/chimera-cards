package api

import (
	"net/http"

	"github.com/ericogr/chimera-cards/internal/constants"
	"github.com/ericogr/chimera-cards/internal/game"
	"github.com/ericogr/chimera-cards/internal/service"

	"github.com/gin-gonic/gin"
)

type CreateHybridSpec struct {
	EntityIDs []uint `json:"entity_ids"`
	// SelectedEntityID must be one of EntityIDs and will define the special
	// ability available for this hybrid during combat.
	SelectedEntityID uint `json:"selected_entity_id"`
}

type CreateHybridsPayload struct {
	Hybrid1 CreateHybridSpec `json:"hybrid1"`
	Hybrid2 CreateHybridSpec `json:"hybrid2"`
}

func sumEntityStats(entities []game.Entity) (hitPoints, attack, defense, agility, energy int) {
	for _, a := range entities {
		hitPoints += a.HitPoints
		attack += a.Attack
		defense += a.Defense
		agility += a.Agility
		energy += a.Energy
	}
	return
}

// Vigor cost per entity is configurable via the chimera_config.json and
// stored on each Entity as `VigorCost`.

// CreateHybrids stores two hybrids for a player in a game.
func (h *GameHandler) CreateHybrids(c *gin.Context) {
	// The path param is the game's join code
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
	var req CreateHybridsPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidRequest})
		return
	}

	// Derive player UUID from the authenticated session and the game's players
	userEmail, _ := c.Get("userEmail")
	emailStr, _ := userEmail.(string)
	if emailStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{constants.JSONKeyError: constants.ErrAuthRequired})
		return
	}
	// Ensure session user is a participant and pass their email to the service
	found := false
	for i := range g.Players {
		if g.Players[i].PlayerEmail == emailStr {
			found = true
			break
		}
	}
	if !found {
		c.JSON(http.StatusForbidden, gin.H{constants.JSONKeyError: constants.ErrPlayerNotPartOfThisGame})
		return
	}

	srvReq := service.CreateHybridsRequest{
		PlayerEmail: emailStr,
		Hybrid1:     service.CreateHybridSpec{EntityIDs: req.Hybrid1.EntityIDs, SelectedEntityID: req.Hybrid1.SelectedEntityID},
		Hybrid2:     service.CreateHybridSpec{EntityIDs: req.Hybrid2.EntityIDs, SelectedEntityID: req.Hybrid2.SelectedEntityID},
	}

	if err := service.CreateHybrids(h.repo, g.ID, srvReq); err != nil {
		switch err {
		case service.ErrGameNotFound:
			c.JSON(http.StatusNotFound, gin.H{constants.JSONKeyError: constants.ErrGameNotFound})
			return
		case service.ErrPlayerNotFound:
			c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrPlayerNotPartOfThisGame})
			return
		case service.ErrHybridsAlreadyCreated:
			c.JSON(http.StatusConflict, gin.H{constants.JSONKeyError: constants.ErrHybridsAlreadyCreated})
			return
		case service.ErrInvalidHybridCount, service.ErrInvalidSelectedAbility, service.ErrEntityReused, service.ErrInvalidEntities:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedSaveHybrids})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Hybrids created"})
}
