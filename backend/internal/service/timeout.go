package service

import (
	"time"

	"github.com/ericogr/chimera-cards/internal/engine"
	"github.com/ericogr/chimera-cards/internal/game"
	"github.com/ericogr/chimera-cards/internal/logging"
)

// HandleTimedOutGame applies timeout resolution for a single game.
// Behavior:
// - both players didn't submit -> finish match with no winner
// - exactly one player didn't submit -> auto-submit rest for that player
// The function uses SubmitAction when the repo implements GameRepo; otherwise
// it falls back to inline resolution via engine.ResolveRound.
func HandleTimedOutGame(repo interface {
	GetGameByID(uint) (*game.Game, error)
	UpdateGame(*game.Game) error
	UpdateStatsOnGameEnd(*game.Game, string) error
}, gg *game.Game, actionTimeout time.Duration) error {
	if gg.Status != game.StatusInProgress || gg.Phase != game.PhasePlanning {
		return nil
	}

	if len(gg.Players) != 2 {
		gg.Status = game.StatusFinished
		gg.Phase = game.PhaseResolved
		gg.Winner = ""
		gg.Message = "Match ended due to inactivity"
		gg.LastRoundSummary = "no resolution was reached due to inactivity."
		gg.StatsCounted = true
		gg.ActionDeadline = time.Time{}
		return repo.UpdateGame(gg)
	}

	p1 := &gg.Players[0]
	p2 := &gg.Players[1]
	p1Submitted := p1.HasSubmittedAction
	p2Submitted := p2.HasSubmittedAction

	switch {
	case !p1Submitted && !p2Submitted:
		gg.Status = game.StatusFinished
		gg.Phase = game.PhaseResolved
		gg.Winner = ""
		gg.Message = "Match ended due to inactivity"
		gg.LastRoundSummary = "Round timed out: both players failed to submit actions within the allotted time."
		gg.StatsCounted = true
		gg.ActionDeadline = time.Time{}
		logging.Info("both players timed out; finishing game", nil)
		return repo.UpdateGame(gg)
	case p1Submitted && !p2Submitted:
		logging.Info("auto-submitting rest for inactive player (p2)", nil)
		// try to use SubmitAction path if repo implements GameRepo
		if gr, ok := repo.(GameRepo); ok {
			_, _, err := SubmitAction(gr, gg.ID, p2.PlayerEmail, game.PendingActionRest, 0, actionTimeout)
			if err != nil {
				logging.Error("SubmitAction auto-rest failed; falling back", err, nil)
			}
			return nil
		}
		// fallback inline
		p2.HasSubmittedAction = true
		p2.PendingActionType = game.PendingActionRest
		p2.PendingActionEntityID = nil
		engine.ResolveRound(gg)
		if gg.Status == game.StatusFinished {
			if !gg.StatsCounted {
				_ = repo.UpdateStatsOnGameEnd(gg, "")
				gg.StatsCounted = true
			}
		} else {
			gg.ActionDeadline = time.Now().Add(actionTimeout)
		}
		return repo.UpdateGame(gg)
	case !p1Submitted && p2Submitted:
		logging.Info("auto-submitting rest for inactive player (p1)", nil)
		if gr, ok := repo.(GameRepo); ok {
			_, _, err := SubmitAction(gr, gg.ID, p1.PlayerEmail, game.PendingActionRest, 0, actionTimeout)
			if err != nil {
				logging.Error("SubmitAction auto-rest failed; falling back", err, nil)
			}
			return nil
		}
		p1.HasSubmittedAction = true
		p1.PendingActionType = game.PendingActionRest
		p1.PendingActionEntityID = nil
		engine.ResolveRound(gg)
		if gg.Status == game.StatusFinished {
			if !gg.StatsCounted {
				_ = repo.UpdateStatsOnGameEnd(gg, "")
				gg.StatsCounted = true
			}
		} else {
			gg.ActionDeadline = time.Now().Add(actionTimeout)
		}
		return repo.UpdateGame(gg)
	default:
		// shouldn't happen
		return nil
	}
}
