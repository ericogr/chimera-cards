package engine

import (
	"math/rand"
	"testing"

	"github.com/ericogr/chimera-cards/internal/game"
)

func TestResolveRound_BasicAttacks(t *testing.T) {
	rand.New(rand.NewSource(1))
	g := &game.Game{Players: []game.Player{
		{PlayerUUID: "p1", PlayerName: "P1", Hybrids: []game.Hybrid{{Name: "H1", BaseHitPoints: 10, CurrentHitPoints: 10, BaseAttack: 10, CurrentAttack: 10, BaseDefense: 1, CurrentDefense: 1, BaseAgility: 5, CurrentAgility: 5, IsActive: true}}},
		{PlayerUUID: "p2", PlayerName: "P2", Hybrids: []game.Hybrid{{Name: "H2", BaseHitPoints: 10, CurrentHitPoints: 10, BaseAttack: 1, CurrentAttack: 1, BaseDefense: 1, CurrentDefense: 1, BaseAgility: 1, CurrentAgility: 1, IsActive: true}}},
	}}
	g.Status = game.StatusInProgress
	g.RoundCount = 1
	g.Players[0].PendingActionType = game.PendingActionBasicAttack
	g.Players[0].HasSubmittedAction = true
	g.Players[1].PendingActionType = game.PendingActionBasicAttack
	g.Players[1].HasSubmittedAction = true
	oldRound := g.RoundCount

	ResolveRound(g)

	if g.Players[1].Hybrids[0].CurrentHitPoints >= 10 {
		t.Fatalf("expected player 2 hybrid to take damage, got PV=%d", g.Players[1].Hybrids[0].CurrentHitPoints)
	}
	if g.RoundCount <= oldRound {
		t.Fatalf("expected round to increment, got %d", g.RoundCount)
	}
}
