package api

import (
	"net/http"
	"strconv"

	"github.com/ericogr/quimera-cards/internal/constants"
	"github.com/ericogr/quimera-cards/internal/game"
	"github.com/ericogr/quimera-cards/internal/service"

	"github.com/gin-gonic/gin"
)

type CreateHybridSpec struct {
	Name      string `json:"name"`
	AnimalIDs []uint `json:"animal_ids"`
	// SelectedAnimalID must be one of AnimalIDs and will define the special
	// ability available for this hybrid during combat.
	SelectedAnimalID uint `json:"selected_animal_id"`
}

type CreateHybridsPayload struct {
	PlayerUUID string           `json:"player_uuid"`
	Hybrid1    CreateHybridSpec `json:"hybrid1"`
	Hybrid2    CreateHybridSpec `json:"hybrid2"`
}

func sumAnimalStats(animals []game.Animal) (hitPoints, attack, defense, agility, energy int) {
	for _, a := range animals {
		hitPoints += a.HitPoints
		attack += a.Attack
		defense += a.Defense
		agility += a.Agility
		energy += a.Energy
	}
	return
}

// Vigor cost per animal is configurable via the chimera_config.json and
// stored on each Animal as `VigorCost`.

// CreateHybrids stores two hybrids for a player in a game.
func (h *GameHandler) CreateHybrids(c *gin.Context) {
	gameID, err := strconv.Atoi(c.Param("gameID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidGameID})
		return
	}
	var req CreateHybridsPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{constants.JSONKeyError: constants.ErrInvalidRequest})
		return
	}

	srvReq := service.CreateHybridsRequest{
		PlayerUUID: req.PlayerUUID,
		Hybrid1:    service.CreateHybridSpec{AnimalIDs: req.Hybrid1.AnimalIDs, SelectedAnimalID: req.Hybrid1.SelectedAnimalID},
		Hybrid2:    service.CreateHybridSpec{AnimalIDs: req.Hybrid2.AnimalIDs, SelectedAnimalID: req.Hybrid2.SelectedAnimalID},
	}

	if err := service.CreateHybrids(h.repo, uint(gameID), srvReq); err != nil {
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
		case service.ErrInvalidHybridCount, service.ErrInvalidSelectedAbility, service.ErrAnimalReused, service.ErrInvalidAnimals:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{constants.JSONKeyError: constants.ErrFailedSaveHybrids})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Hybrids created"})
}
