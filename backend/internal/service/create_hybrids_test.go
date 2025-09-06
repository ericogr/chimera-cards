package service

import (
	"reflect"
	"testing"

	"github.com/ericogr/chimera-cards/internal/game"
)

type mockRepo struct {
	games       map[uint]*game.Game
	entities    map[uint]game.Entity
	updatedGame *game.Game
}

func (m *mockRepo) GetGameByID(id uint) (*game.Game, error) {
	if g, ok := m.games[id]; ok {
		return g, nil
	}
	return nil, ErrGameNotFound
}

func (m *mockRepo) GetEntitiesByIDs(ids []uint) ([]game.Entity, error) {
	res := make([]game.Entity, 0, len(ids))
	for _, id := range ids {
		a, ok := m.entities[id]
		if !ok {
			return nil, ErrInvalidEntities
		}
		res = append(res, a)
	}
	return res, nil
}

func (m *mockRepo) UpdateGame(g *game.Game) error {
	m.updatedGame = g
	return nil
}

func (m *mockRepo) UpdateStatsOnGameEnd(g *game.Game, resignedEmail string) error {
	// noop for tests
	return nil
}

func TestCreateHybridsSuccess(t *testing.T) {
// Prepare entities
	entities := map[uint]game.Entity{
		1: {Model: game.Entity{}.Model, Name: "Lion", HitPoints: 4, Attack: 8, Defense: 4, Agility: 5, Energy: 0},
		2: {Model: game.Entity{}.Model, Name: "Raven", HitPoints: 2, Attack: 3, Defense: 3, Agility: 7, Energy: 1},
		3: {Model: game.Entity{}.Model, Name: "Wolf", HitPoints: 4, Attack: 5, Defense: 4, Agility: 6, Energy: 1},
		4: {Model: game.Entity{}.Model, Name: "Octopus", HitPoints: 5, Attack: 2, Defense: 5, Agility: 4, Energy: 1},
	}

	g := &game.Game{Players: []game.Player{{PlayerUUID: "p1"}}}
	mr := &mockRepo{games: map[uint]*game.Game{42: g}, entities: entities}

	req := CreateHybridsRequest{
		PlayerUUID: "p1",
		Hybrid1:    CreateHybridSpec{EntityIDs: []uint{1, 2}, SelectedEntityID: 1},
		Hybrid2:    CreateHybridSpec{EntityIDs: []uint{3, 4}, SelectedEntityID: 3},
	}

	if err := CreateHybrids(mr, 42, req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mr.updatedGame == nil {
		t.Fatalf("expected updated game to be saved")
	}
	p := mr.updatedGame.Players[0]
	if !p.HasCreated {
		t.Fatalf("expected HasCreated true")
	}
	if len(p.Hybrids) != 2 {
		t.Fatalf("expected 2 hybrids, got %d", len(p.Hybrids))
	}
	// Check stats for first hybrid: Lion + Raven -> PV 4+2=6
	if p.Hybrids[0].BaseHitPoints != 6 {
		t.Fatalf("expected base PV 6, got %d", p.Hybrids[0].BaseHitPoints)
	}
	// Check names are concatenated and sorted
	expectedName := "Lion + Raven"
	if p.Hybrids[0].Name != expectedName && p.Hybrids[0].Name != "Raven + Lion" {
		// Allow both orders depending on sorting
		t.Fatalf("unexpected hybrid name: %q", p.Hybrids[0].Name)
	}
}

func TestCreateHybrids_ReusedAnimal(t *testing.T) {
	entities := map[uint]game.Entity{
		1: {Model: game.Entity{}.Model, Name: "Lion", HitPoints: 4, Attack: 8, Defense: 4, Agility: 5, Energy: 0},
		2: {Model: game.Entity{}.Model, Name: "Raven", HitPoints: 2, Attack: 3, Defense: 3, Agility: 7, Energy: 1},
		3: {Model: game.Entity{}.Model, Name: "Wolf", HitPoints: 4, Attack: 5, Defense: 4, Agility: 6, Energy: 1},
	}
	g := &game.Game{Players: []game.Player{{PlayerUUID: "p1"}}}
	mr := &mockRepo{games: map[uint]*game.Game{100: g}, entities: entities}

	req := CreateHybridsRequest{
		PlayerUUID: "p1",
		Hybrid1:    CreateHybridSpec{EntityIDs: []uint{1, 2}, SelectedEntityID: 1},
		Hybrid2:    CreateHybridSpec{EntityIDs: []uint{2, 3}, SelectedEntityID: 3},
	}

	err := CreateHybrids(mr, 100, req)
    if err == nil {
        t.Fatalf("expected error for reused entity, got nil")
    }
	if !reflect.DeepEqual(err, ErrEntityReused) {
		t.Fatalf("expected ErrEntityReused, got %v", err)
	}
}
