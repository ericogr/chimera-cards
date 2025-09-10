package service

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ericogr/chimera-cards/internal/game"
)

type mockRepoSA struct {
	games       map[uint]*game.Game
	updatedGame *game.Game
	statsCalled bool
}

func (m *mockRepoSA) GetGameByID(id uint) (*game.Game, error) {
	if g, ok := m.games[id]; ok {
		return g, nil
	}
	return nil, ErrGameNotFound
}

func (m *mockRepoSA) GetEntitiesByIDs(ids []uint) ([]game.Entity, error) {
	return nil, nil
}

func (m *mockRepoSA) UpdateGame(g *game.Game) error {
	m.updatedGame = g
	return nil
}

func (m *mockRepoSA) UpdateStatsOnGameEnd(g *game.Game, resignedEmail string) error {
	m.statsCalled = true
	return nil
}

func TestSubmitAction_ResolvesRound(t *testing.T) {
	rand.Seed(1)
	g := &game.Game{Players: []game.Player{
		{PlayerUUID: "p1", PlayerName: "P1", Hybrids: []game.Hybrid{{Name: "H1", BaseHitPoints: 10, CurrentHitPoints: 10, BaseAttack: 10, CurrentAttack: 10, BaseDefense: 1, CurrentDefense: 1, BaseAgility: 5, CurrentAgility: 5, IsActive: true}}},
		{PlayerUUID: "p2", PlayerName: "P2", Hybrids: []game.Hybrid{{Name: "H2", BaseHitPoints: 10, CurrentHitPoints: 10, BaseAttack: 1, CurrentAttack: 1, BaseDefense: 1, CurrentDefense: 1, BaseAgility: 1, CurrentAgility: 1, IsActive: true}}},
	}}
	g.RoundCount = 1
	g.Status = "in_progress"
	g.Phase = "planning"
	mr := &mockRepoSA{games: map[uint]*game.Game{7: g}}

	// First player submits
	_, resolved, err := SubmitAction(mr, 7, "p1", game.PendingActionBasicAttack, 0, 1*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved {
		t.Fatalf("round should not be resolved after only one submission")
	}

	// Second player submits -> should resolve
	g2, resolved, err := SubmitAction(mr, 7, "p2", game.PendingActionBasicAttack, 0, 1*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resolved {
		t.Fatalf("expected round to be resolved")
	}
	if g2.RoundCount != 2 {
		t.Fatalf("expected RoundCount=2 after resolution, got %d", g2.RoundCount)
	}
}
