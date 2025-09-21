package service

import (
	"testing"
	"time"

	"github.com/ericogr/chimera-cards/internal/game"
)

type mockRepoTimeout struct {
	g *game.Game
}

func (m *mockRepoTimeout) GetGameByID(id uint) (*game.Game, error)                       { return m.g, nil }
func (m *mockRepoTimeout) UpdateGame(g *game.Game) error                                 { m.g = g; return nil }
func (m *mockRepoTimeout) UpdateStatsOnGameEnd(g *game.Game, resignedEmail string) error { return nil }

func TestHandleTimedOutGame_BothMiss(t *testing.T) {
	g := &game.Game{Status: game.StatusInProgress, Phase: game.PhasePlanning, ActionDeadline: time.Now().Add(-time.Minute), Players: []game.Player{{PlayerName: "A"}, {PlayerName: "B"}}}
	mr := &mockRepoTimeout{g: g}
	if err := HandleTimedOutGame(mr, g, 1*time.Minute); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mr.g.Status != game.StatusFinished {
		t.Fatalf("expected finished status, got %v", mr.g.Status)
	}
}

func TestHandleTimedOutGame_OneMiss(t *testing.T) {
	g := &game.Game{Status: game.StatusInProgress, Phase: game.PhasePlanning, ActionDeadline: time.Now().Add(-time.Minute)}
	p1 := game.Player{PlayerName: "A", PlayerEmail: "a@e.com", HasSubmittedAction: true}
	p2 := game.Player{PlayerName: "B", PlayerEmail: "b@e.com", HasSubmittedAction: false, Hybrids: []game.Hybrid{{BaseHitPoints: 5, CurrentHitPoints: 5, IsActive: true}}}
	g.Players = []game.Player{p1, p2}
	mr := &mockRepoTimeout{g: g}
	if err := HandleTimedOutGame(mr, g, 1*time.Minute); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mr.g.Players[1].HasSubmittedAction {
		t.Fatalf("expected player 2 to be auto-submitted as rest")
	}
}
