package service

import (
	"errors"
	"sort"
	"strings"

	"github.com/ericogr/quimera-cards/internal/game"
)

// GameRepo is the minimal repository interface required by CreateHybrids.
// Using a small interface simplifies testing.
type GameRepo interface {
	GetGameByID(id uint) (*game.Game, error)
	GetAnimalsByIDs(ids []uint) ([]game.Animal, error)
	UpdateGame(g *game.Game) error
	UpdateStatsOnGameEnd(g *game.Game, resignedEmail string) error
}

type CreateHybridSpec struct {
	AnimalIDs        []uint
	SelectedAnimalID uint
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
	ErrInvalidHybridCount     = errors.New("each hybrid must have 2 or 3 animals")
	ErrInvalidSelectedAbility = errors.New("selected ability must reference one of the hybrid's animals")
	ErrAnimalReused           = errors.New("the same animal cannot be reused across hybrids")
	ErrInvalidAnimals         = errors.New("invalid animals for hybrid")
)

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
	if !validCount(len(req.Hybrid1.AnimalIDs)) || !validCount(len(req.Hybrid2.AnimalIDs)) {
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
	if !contains(req.Hybrid1.AnimalIDs, req.Hybrid1.SelectedAnimalID) || !contains(req.Hybrid2.AnimalIDs, req.Hybrid2.SelectedAnimalID) {
		return ErrInvalidSelectedAbility
	}

	used := map[uint]bool{}
	for _, id := range append(req.Hybrid1.AnimalIDs, req.Hybrid2.AnimalIDs...) {
		if used[id] {
			return ErrAnimalReused
		}
		used[id] = true
	}

	animals1, err := repo.GetAnimalsByIDs(req.Hybrid1.AnimalIDs)
	if err != nil || len(animals1) != len(req.Hybrid1.AnimalIDs) || !validCount(len(animals1)) {
		return ErrInvalidAnimals
	}
	animals2, err := repo.GetAnimalsByIDs(req.Hybrid2.AnimalIDs)
	if err != nil || len(animals2) != len(req.Hybrid2.AnimalIDs) || !validCount(len(animals2)) {
		return ErrInvalidAnimals
	}

	hp1, at1, def1, agility1, energy1 := sumAnimalStats(animals1)
	hp2, at2, def2, agility2, energy2 := sumAnimalStats(animals2)

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

	computeName := func(an []game.Animal) string {
		names := make([]string, len(an))
		for i := range an {
			names[i] = an[i].Name
		}
		sort.Slice(names, func(i, j int) bool { return strings.ToLower(names[i]) < strings.ToLower(names[j]) })
		return strings.Join(names, " + ")
	}

	h1 := game.Hybrid{
		Name:          computeName(animals1),
		BaseAnimals:   animals1,
		BaseHitPoints: hp1,
		BaseAttack:    at1,
		BaseDefense:   def1,
		BaseAgility:   agility1,
		BaseEnergy:    energy1,
	}
	sel1 := req.Hybrid1.SelectedAnimalID
	h1.SelectedAbilityAnimalID = &sel1

	h2 := game.Hybrid{
		Name:          computeName(animals2),
		BaseAnimals:   animals2,
		BaseHitPoints: hp2,
		BaseAttack:    at2,
		BaseDefense:   def2,
		BaseAgility:   agility2,
		BaseEnergy:    energy2,
	}
	sel2 := req.Hybrid2.SelectedAnimalID
	h2.SelectedAbilityAnimalID = &sel2

	p.Hybrids = []game.Hybrid{h1, h2}
	p.HasCreated = true
	g.Message = "Hybrids created for a player."

	if err := repo.UpdateGame(g); err != nil {
		return err
	}
	return nil
}
