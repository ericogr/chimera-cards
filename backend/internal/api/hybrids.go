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
	PlayerUUID string           `json:"player_uuid"`
	Hybrid1    CreateHybridSpec `json:"hybrid1"`
	Hybrid2    CreateHybridSpec `json:"hybrid2"`
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

	srvReq := service.CreateHybridsRequest{
		PlayerUUID: req.PlayerUUID,
		Hybrid1:    service.CreateHybridSpec{EntityIDs: req.Hybrid1.EntityIDs, SelectedEntityID: req.Hybrid1.SelectedEntityID},
		Hybrid2:    service.CreateHybridSpec{EntityIDs: req.Hybrid2.EntityIDs, SelectedEntityID: req.Hybrid2.SelectedEntityID},
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
