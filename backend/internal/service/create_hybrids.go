package service

import (
	"errors"
	"sort"
	"strings"

	"github.com/ericogr/chimera-cards/internal/game"
)

// GameRepo is the minimal repository interface required by CreateHybrids.
// Using a small interface simplifies testing.
type GameRepo interface {
	GetGameByID(id uint) (*game.Game, error)
	GetEntitiesByIDs(ids []uint) ([]game.Entity, error)
	UpdateGame(g *game.Game) error
	UpdateStatsOnGameEnd(g *game.Game, resignedEmail string) error
}

type CreateHybridSpec struct {
	EntityIDs        []uint
	SelectedEntityID uint
}

type CreateHybridsRequest struct {
	PlayerUUID string
	Hybrid1    CreateHybridSpec
	Hybrid2    CreateHybridSpec
}

var (
	ErrGameNotFound           = errors.New("game not found")
	ErrPlayerNotFound         = errors.New("player not part of game")
	ErrHybridsAlreadyCreated  = errors.New("hybrids already created")
	ErrInvalidHybridCount     = errors.New("each hybrid must have 2 or 3 entities")
	ErrInvalidSelectedAbility = errors.New("selected ability must reference one of the hybrid's entities")
	ErrEntityReused           = errors.New("the same entity cannot be reused across hybrids")
	ErrInvalidEntities        = errors.New("invalid entities for hybrid")
)

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

// CreateHybrids builds and stores two hybrids for a player inside a game.
// It performs all validation and persists the updated game via the repo.
func CreateHybrids(repo GameRepo, gameID uint, req CreateHybridsRequest) error {
	g, err := repo.GetGameByID(gameID)
	if err != nil || g == nil {
		return ErrGameNotFound
	}

	// Find player
	var p *game.Player
	for i := range g.Players {
		if g.Players[i].PlayerUUID == req.PlayerUUID {
			p = &g.Players[i]
			break
		}
	}
	if p == nil {
		return ErrPlayerNotFound
	}
	if p.HasCreated || len(p.Hybrids) > 0 {
		return ErrHybridsAlreadyCreated
	}

	validCount := func(n int) bool { return n >= 2 && n <= 3 }
	if !validCount(len(req.Hybrid1.EntityIDs)) || !validCount(len(req.Hybrid2.EntityIDs)) {
		return ErrInvalidHybridCount
	}

	contains := func(ids []uint, id uint) bool {
		for _, x := range ids {
			if x == id {
				return true
			}
		}
		return false
	}
	if !contains(req.Hybrid1.EntityIDs, req.Hybrid1.SelectedEntityID) || !contains(req.Hybrid2.EntityIDs, req.Hybrid2.SelectedEntityID) {
		return ErrInvalidSelectedAbility
	}

	used := map[uint]bool{}
	for _, id := range append(req.Hybrid1.EntityIDs, req.Hybrid2.EntityIDs...) {
		if used[id] {
			return ErrEntityReused
		}
		used[id] = true
	}

	entities1, err := repo.GetEntitiesByIDs(req.Hybrid1.EntityIDs)
	if err != nil || len(entities1) != len(req.Hybrid1.EntityIDs) || !validCount(len(entities1)) {
		return ErrInvalidEntities
	}
	entities2, err := repo.GetEntitiesByIDs(req.Hybrid2.EntityIDs)
	if err != nil || len(entities2) != len(req.Hybrid2.EntityIDs) || !validCount(len(entities2)) {
		return ErrInvalidEntities
	}

	hp1, at1, def1, agility1, energy1 := sumEntityStats(entities1)
	hp2, at2, def2, agility2, energy2 := sumEntityStats(entities2)

	clamp := func(x, lo, hi int) int {
		if x < lo {
			return lo
		}
		if x > hi {
			return hi
		}
		return x
	}
	energy1 = clamp(energy1, 1, 3)
	energy2 = clamp(energy2, 1, 3)

	computeName := func(an []game.Entity) string {
		names := make([]string, len(an))
		for i := range an {
			names[i] = an[i].Name
		}
		sort.Slice(names, func(i, j int) bool { return strings.ToLower(names[i]) < strings.ToLower(names[j]) })
		return strings.Join(names, " + ")
	}

	h1 := game.Hybrid{
		Name:          computeName(entities1),
		BaseEntities:  entities1,
		BaseHitPoints: hp1,
		BaseAttack:    at1,
		BaseDefense:   def1,
		BaseAgility:   agility1,
		BaseEnergy:    energy1,
	}
	sel1 := req.Hybrid1.SelectedEntityID
	h1.SelectedAbilityEntityID = &sel1

	h2 := game.Hybrid{
		Name:          computeName(entities2),
		BaseEntities:  entities2,
		BaseHitPoints: hp2,
		BaseAttack:    at2,
		BaseDefense:   def2,
		BaseAgility:   agility2,
		BaseEnergy:    energy2,
	}
	sel2 := req.Hybrid2.SelectedEntityID
	h2.SelectedAbilityEntityID = &sel2

	p.Hybrids = []game.Hybrid{h1, h2}
	p.HasCreated = true
	g.Message = "Hybrids created for a player."

	if err := repo.UpdateGame(g); err != nil {
		return err
	}
	return nil
}
